package server

import (
	"L4.2/internal/cut"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// NetworkServer сервер
type NetworkServer struct {
	id           int
	totalServers int
	address      string
	cutConfig    cut.Config

	// Сетевые соединения
	listener    net.Listener
	connections map[int]net.Conn
	connMu      sync.RWMutex

	// Обработка данных
	linesChan   chan string
	localResult []string

	// Голосование
	votesReceived map[int]bool
	votesMu       sync.RWMutex

	// Состояние
	state   string // "idle", "processing", "completed"
	stateMu sync.RWMutex

	// Управление
	jobID    string
	quitChan chan struct{}
	wg       sync.WaitGroup
}

// MessageType тип сообщения
type MessageType string

const (
	// MsgVote голос
	MsgVote MessageType = "VOTE"
	// MsgResult результат
	MsgResult MessageType = "RESULT"
	// MsgCommit подтверждение
	MsgCommit MessageType = "COMMIT"
	// MsgReady готов
	MsgReady MessageType = "READY"
)

// Message сообщение
type Message struct {
	Type      MessageType `json:"type"`
	ServerID  int         `json:"server_id"`
	JobID     string      `json:"job_id"`
	Status    bool        `json:"status"`
	Lines     []string    `json:"lines,omitempty"`
	LineCount int         `json:"line_count"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewNetworkServer создание экземпляра сервера
func NewNetworkServer(id, totalServers int, address string, cutConfig cut.Config) *NetworkServer {
	return &NetworkServer{
		id:            id,
		totalServers:  totalServers,
		address:       address,
		cutConfig:     cutConfig,
		connections:   make(map[int]net.Conn),
		linesChan:     make(chan string, 10000),
		votesReceived: make(map[int]bool),
		state:         "idle",
		jobID:         fmt.Sprintf("job_%d", time.Now().UnixNano()),
		quitChan:      make(chan struct{}),
	}
}

// Start Запуск TCP сервера для приема соединений
func (s *NetworkServer) Start() error {
	// Запускаем listener
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	s.listener = listener

	fmt.Printf("[Server %d] Listening on %s\n", s.id, s.address)

	// Принимаем входящие соединения
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Прием входящих соединений от других серверов
func (s *NetworkServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quitChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}

			// Обрабатываем соединение в горутине
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

// Обработка одного соединения
func (s *NetworkServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			return // Соединение закрыто или ошибка
		}

		s.handleMessage(msg)
	}
}

// ConnectToPeers Подключение к другим серверам
func (s *NetworkServer) ConnectToPeers(peerAddresses map[int]string) error {
	for peerID, address := range peerAddresses {
		if peerID == s.id {
			continue
		}

		fmt.Printf("[Server %d] Connecting to Server %d at %s\n", s.id, peerID, address)

		conn, err := net.Dial("tcp", address)
		if err != nil {
			fmt.Printf("[Server %d] Failed to connect to Server %d: %v\n", s.id, peerID, err)
			continue
		}

		s.connMu.Lock()
		s.connections[peerID] = conn
		s.connMu.Unlock()

		// Запускаем горутину для чтения ответов от этого пира
		s.wg.Add(1)
		go s.readFromPeer(peerID, conn)
	}

	return nil
}

// Чтение сообщений от конкретного пира
func (s *NetworkServer) readFromPeer(peerID int, conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			fmt.Printf("[Server %d] Connection to Server %d closed\n", s.id, peerID)
			s.connMu.Lock()
			delete(s.connections, peerID)
			s.connMu.Unlock()
			return
		}

		s.handleMessage(msg)
	}
}

// Обработка входящего сообщения
func (s *NetworkServer) handleMessage(msg Message) {
	switch msg.Type {
	case MsgVote:
		s.votesMu.Lock()
		s.votesReceived[msg.ServerID] = msg.Status
		s.votesMu.Unlock()
		fmt.Printf("[Server %d] Received VOTE from Server %d: status=%v\n",
			s.id, msg.ServerID, msg.Status)

	case MsgResult:
		fmt.Printf("[Server %d] Received RESULT from Server %d: %d lines\n",
			s.id, msg.ServerID, msg.LineCount)

	case MsgCommit:
		fmt.Printf("[Server %d] Received COMMIT from Server %d\n", s.id, msg.ServerID)
		select {
		case <-s.quitChan:
		default:
			close(s.quitChan)
		}

	case MsgReady:
		fmt.Printf("[Server %d] Received READY from Server %d\n", s.id, msg.ServerID)
	}
}

// SendToPeer Отправка сообщения конкретному серверу
func (s *NetworkServer) SendToPeer(peerID int, msg Message) error {
	s.connMu.RLock()
	conn, ok := s.connections[peerID]
	s.connMu.RUnlock()

	if !ok {
		return fmt.Errorf("no connection to peer %d", peerID)
	}

	encoder := json.NewEncoder(conn)
	return encoder.Encode(msg)
}

// Broadcast Рассылка сообщения всем серверам
func (s *NetworkServer) Broadcast(msg Message) {
	s.connMu.RLock()
	peers := make([]int, 0, len(s.connections))
	for id := range s.connections {
		peers = append(peers, id)
	}
	s.connMu.RUnlock()

	for _, peerID := range peers {
		if err := s.SendToPeer(peerID, msg); err != nil {
			fmt.Printf("[Server %d] Failed to send to Server %d: %v\n", s.id, peerID, err)
		}
	}
}

// ReceiveData Получение данных для обработки
func (s *NetworkServer) ReceiveData(allLines []string) {
	go func() {
		for i, line := range allLines {
			// Шардирование по индексу строки
			if i%s.totalServers == s.id {
				s.linesChan <- line
			}
		}
		close(s.linesChan)
		fmt.Printf("[Server %d] Received %d lines for processing\n",
			s.id, len(s.linesChan))
	}()
}

// ProcessData Обработка данных
func (s *NetworkServer) ProcessData() {
	s.stateMu.Lock()
	s.state = "processing"
	s.stateMu.Unlock()

	var myLines []string
	for line := range s.linesChan {
		myLines = append(myLines, line)
	}

	fmt.Printf("[Server %d] Processing %d lines...\n", s.id, len(myLines))

	// Применяем cut логику
	results := cut.ProcessBatch(myLines, s.cutConfig)
	s.localResult = results

	s.stateMu.Lock()
	s.state = "completed"
	s.stateMu.Unlock()

	fmt.Printf("[Server %d] Processed %d lines\n", s.id, len(results))

	// Рассылаем голос
	s.Broadcast(Message{
		Type:      MsgVote,
		ServerID:  s.id,
		JobID:     s.jobID,
		Status:    true,
		LineCount: len(results),
		Timestamp: time.Now(),
	})

	// Рассылаем результат
	s.Broadcast(Message{
		Type:      MsgResult,
		ServerID:  s.id,
		JobID:     s.jobID,
		Lines:     results,
		LineCount: len(results),
		Timestamp: time.Now(),
	})
}

// WaitForQuorum Ожидание кворума
func (s *NetworkServer) WaitForQuorum() {
	quorumSize := s.totalServers/2 + 1
	fmt.Printf("[Server %d] Waiting for quorum (need %d/%d votes)\n",
		s.id, quorumSize, s.totalServers)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ticker.C:
			s.votesMu.RLock()
			successCount := 0
			for _, status := range s.votesReceived {
				if status {
					successCount++
				}
			}
			s.votesMu.RUnlock()

			s.stateMu.RLock()
			if s.state == "completed" {
				successCount++
			}
			s.stateMu.RUnlock()

			if successCount >= quorumSize {
				fmt.Printf("\n[Server %d] ✓ QUORUM REACHED! (%d/%d votes)\n",
					s.id, successCount, quorumSize)
				s.onQuorumReached()
				return
			}

		case <-timeout:
			fmt.Printf("[Server %d] ✗ Timeout waiting for quorum\n", s.id)
			return

		case <-s.quitChan:
			return
		}
	}
}

// Когда кворум достигнут
func (s *NetworkServer) onQuorumReached() {
	fmt.Printf("\n[Server %d] ===== FINAL RESULT =====\n", s.id)
	fmt.Printf("Total lines processed: %d\n", len(s.localResult))

	showLines := 20
	if len(s.localResult) < showLines {
		showLines = len(s.localResult)
	}

	fmt.Printf("First %d lines:\n", showLines)
	for i := 0; i < showLines; i++ {
		fmt.Printf("  %d: %s\n", i+1, s.localResult[i])
	}

	if len(s.localResult) > showLines {
		fmt.Printf("  ... and %d more lines\n", len(s.localResult)-showLines)
	}

	// Рассылаем commit
	s.Broadcast(Message{
		Type:      MsgCommit,
		ServerID:  s.id,
		JobID:     s.jobID,
		Timestamp: time.Now(),
	})
}

// Stop Остановка сервера
func (s *NetworkServer) Stop() {
	_, ok := <-s.quitChan
	if ok {
		close(s.quitChan)
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.connMu.Lock()
	for _, conn := range s.connections {
		conn.Close()
	}
	s.connMu.Unlock()
	s.wg.Wait()
	fmt.Printf("[Server %d] Stopped\n", s.id)
}
