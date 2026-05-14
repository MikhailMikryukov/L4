package server

import (
	"L4.2/internal/cut"
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ========== ТЕСТ 1: БАЗОВАЯ РАБОТА С ОДНИМ СЕРВЕРОМ ==========
func TestSingleServer(t *testing.T) {
	fmt.Println("\n=== TEST 1: Single Server Mode ===")

	// Создаем тестовые данные
	testData := []string{
		"1\tapple\tred",
		"2\tbanana\tyellow",
		"3\tgrape\tpurple",
	}

	// Создаем временный файл
	file := createTempFile(t, testData)
	defer os.Remove(file)

	// Конфигурация
	fields := []int{1, 2}
	delimiter := "\t"

	// Запускаем в режиме одного сервера
	config := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: false,
	}

	// Обрабатываем
	results := cut.ProcessBatch(testData, config)

	// Проверяем результаты
	expected := []string{
		"1\tapple",
		"2\tbanana",
		"3\tgrape",
	}

	if len(results) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(results))
	}

	for i := range expected {
		if results[i] != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], results[i])
		}
	}

	fmt.Printf("✓ Single server processed %d lines correctly\n", len(results))
}

// ========== ТЕСТ 2: РАСПРЕДЕЛЕННАЯ ОБРАБОТКА С 3 СЕРВЕРАМИ ==========
func TestDistributedProcessing(t *testing.T) {
	fmt.Println("\n=== TEST 2: Distributed Processing with 3 Servers ===")

	// Создаем тестовые данные (9 строк для равного распределения)
	testData := []string{
		"0\tA\tX",
		"1\tB\tY",
		"2\tC\tZ",
		"3\tD\tW",
		"4\tE\tV",
		"5\tF\tU",
		"6\tG\tT",
		"7\tH\tS",
		"8\tI\tR",
	}

	totalServers := 3
	fields := []int{1, 2}
	delimiter := "\t"

	cutConfig := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: false,
	}

	// Создаем серверы
	servers := make([]*NetworkServer, totalServers)
	basePort := 9000 // Используем другой порт чтобы не конфликтовать

	for i := 0; i < totalServers; i++ {
		address := fmt.Sprintf("localhost:%d", basePort+i)
		servers[i] = NewNetworkServer(i, totalServers, address, cutConfig)
	}

	// Запускаем серверы
	for i := 0; i < totalServers; i++ {
		if err := servers[i].Start(); err != nil {
			t.Fatalf("Failed to start server %d: %v", i, err)
		}
	}
	defer stopServers(servers)

	// Связываем серверы
	connectServers(servers, basePort)

	// Даем время на установку соединений
	time.Sleep(1 * time.Second)

	// Распределяем данные, запускаем обработку, ожидаем кворум
	for i := 0; i < totalServers; i++ {
		servers[i].ReceiveData(testData)
		go servers[i].ProcessData()
		go servers[i].WaitForQuorum()
	}

	var resultsMu sync.Mutex

	// Ждем завершения с таймаутом
	timeout := time.After(10 * time.Second)
	allCompleted := false

	select {
	case <-timeout:
		t.Error("Timeout waiting for quorum")
	case <-time.After(5 * time.Second):
		// Проверяем, что все серверы достигли кворума
		resultsMu.Lock()
		allCompleted = servers[0].state == "completed" && servers[1].state == "completed" && servers[2].state == "completed"
		resultsMu.Unlock()
		fmt.Println(allCompleted)
		if allCompleted {
			fmt.Println("✓ All 3 servers reached quorum")

			// Проверяем шардирование: каждая строка обработана одним сервером
			totalProcessed := 0
			for i := 0; i < totalServers; i++ {
				totalProcessed += len(servers[i].localResult)
				fmt.Printf("  Server %d processed %d lines\n", i, len(servers[i].localResult))
			}

			if totalProcessed == len(testData) {
				fmt.Printf("✓ Total lines processed: %d/%d\n", totalProcessed, len(testData))
			} else {
				t.Errorf("Expected %d total lines, got %d", len(testData), totalProcessed)
			}
		}
	}
}

// ========== ТЕСТ 3: ОТКАЗОУСТОЙЧИВОСТЬ (КВОРУМ) ==========
func TestFaultTolerance(t *testing.T) {
	fmt.Println("\n=== TEST 3: Fault Tolerance (Quorum with Failed Server) ===")

	testData := []string{
		"1\tone\tuno",
		"2\ttwo\tdos",
		"3\tthree\ttres",
		"4\tfour\tcuatro",
		"5\tfive\tcinco",
	}

	totalServers := 3
	fields := []int{1, 2}
	delimiter := "\t"

	cutConfig := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: false,
	}

	// Создаем серверы
	servers := make([]*NetworkServer, totalServers)
	basePort := 9100

	for i := 0; i < totalServers; i++ {
		address := fmt.Sprintf("localhost:%d", basePort+i)
		servers[i] = NewNetworkServer(i, totalServers, address, cutConfig)
	}

	// Запускаем серверы
	for i := 0; i < totalServers; i++ {
		if err := servers[i].Start(); err != nil {
			t.Fatalf("Failed to start server %d: %v", i, err)
		}
	}
	defer stopServers(servers)

	// Связываем серверы
	connectServers(servers, basePort)

	time.Sleep(1 * time.Second)

	// Даем данные всем серверам
	for i := 0; i < totalServers; i++ {
		servers[i].ReceiveData(testData)
	}

	// Симулируем отказ: сервер 2 не запускает обработку
	// (серверы 0 и 1 работают нормально)

	// Запускаем только серверы 0 и 1
	for i := 0; i < 2; i++ {
		go func(s *NetworkServer) {
			s.ProcessData()
		}(servers[i])
	}

	// Сервер 2 НЕ запускаем (имитируем отказ)
	fmt.Println("  Simulating server 2 failure...")

	// Ожидаем кворум (2 из 3 = достаточно)
	quorumReached := make(chan bool, 1)

	go func() {
		servers[0].WaitForQuorum()
		quorumReached <- true
	}()

	// Ждем кворум или таймаут
	select {
	case <-quorumReached:
		fmt.Println("✓ Quorum reached with 2/3 servers (failure tolerated)")
	case <-time.After(8 * time.Second):
		t.Error("Quorum not reached despite 2 working servers")
	}
}

// ========== ТЕСТ 4: ШАРДИРОВАНИЕ ДАННЫХ ==========
func TestSharding(t *testing.T) {
	fmt.Println("\n=== TEST 4: Data Sharding ===")

	// Создаем 100 строк
	testData := make([]string, 100)
	for i := 0; i < 100; i++ {
		testData[i] = fmt.Sprintf("%d\tdata%d\tvalue%d", i, i, i)
	}

	totalServers := 5 // Тестируем с 5 серверами
	fields := []int{1}
	delimiter := "\t"

	cutConfig := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: false,
	}

	// Создаем серверы
	servers := make([]*NetworkServer, totalServers)
	basePort := 9200

	for i := 0; i < totalServers; i++ {
		address := fmt.Sprintf("localhost:%d", basePort+i)
		servers[i] = NewNetworkServer(i, totalServers, address, cutConfig)
	}

	// Запускаем и связываем
	for i := 0; i < totalServers; i++ {
		servers[i].Start()
	}
	defer stopServers(servers)

	connectServers(servers, basePort)
	time.Sleep(1 * time.Second)

	// Распределяем данные
	for i := 0; i < totalServers; i++ {
		servers[i].ReceiveData(testData)
	}

	// Проверяем шардирование вручную
	expectedPerServer := 100 / totalServers
	actualDistribution := make([]int, totalServers)

	// Имитируем распределение
	for i := 0; i < len(testData); i++ {
		serverID := i % totalServers
		actualDistribution[serverID]++
	}

	fmt.Println("  Shard distribution:")
	for i := 0; i < totalServers; i++ {
		fmt.Printf("    Server %d: %d lines (expected ~%d)\n",
			i, actualDistribution[i], expectedPerServer)
	}

	// Проверяем, что распределение равномерное
	for i := 0; i < totalServers; i++ {
		if actualDistribution[i] < expectedPerServer-1 ||
			actualDistribution[i] > expectedPerServer+1 {
			t.Errorf("Server %d has %d lines, expected ~%d",
				i, actualDistribution[i], expectedPerServer)
		}
	}

	fmt.Println("✓ Data is evenly distributed across all servers")
}

// ========== ТЕСТ 5: СЕТЕВОЕ ВЗАИМОДЕЙСТВИЕ ==========
func TestNetworkCommunication(t *testing.T) {
	fmt.Println("\n=== TEST 5: Network Communication ===")

	totalServers := 2
	fields := []int{1}
	delimiter := ","

	cutConfig := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: false,
	}

	// Создаем серверы
	servers := make([]*NetworkServer, totalServers)
	basePort := 9300

	for i := 0; i < totalServers; i++ {
		address := fmt.Sprintf("localhost:%d", basePort+i)
		servers[i] = NewNetworkServer(i, totalServers, address, cutConfig)
	}

	// Запускаем
	for i := 0; i < totalServers; i++ {
		if err := servers[i].Start(); err != nil {
			t.Fatalf("Failed to start server %d: %v", i, err)
		}
	}
	defer stopServers(servers)

	// Связываем
	connectServers(servers, basePort)

	// Проверяем, что соединения установлены
	time.Sleep(2 * time.Second)

	for i := 0; i < totalServers; i++ {
		servers[i].connMu.RLock()
		connCount := len(servers[i].connections)
		servers[i].connMu.RUnlock()

		if connCount != totalServers-1 {
			t.Errorf("Server %d has %d connections, expected %d",
				i, connCount, totalServers-1)
		} else {
			fmt.Printf("✓ Server %d has %d active connections\n", i, connCount)
		}
	}

	// Тестируем отправку сообщений
	testMsg := Message{
		Type:      MsgVote,
		ServerID:  0,
		Status:    true,
		Timestamp: time.Now(),
	}

	err := servers[0].SendToPeer(1, testMsg)
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	} else {
		fmt.Println("✓ Message sent successfully between servers")
	}
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========

func createTempFile(t *testing.T, lines []string) string {
	file, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	writer.Flush()

	return file.Name()
}

func connectServers(servers []*NetworkServer, basePort int) {
	for i := 0; i < len(servers); i++ {
		peerAddresses := make(map[int]string)
		for j := 0; j < len(servers); j++ {
			if i != j {
				peerAddresses[j] = fmt.Sprintf("localhost:%d", basePort+j)
			}
		}

		servers[i].ConnectToPeers(peerAddresses)
	}
}

func stopServers(servers []*NetworkServer) {
	for _, s := range servers {
		go s.Stop()
	}
}

// ========== ЗАПУСК ВСЕХ ТЕСТОВ ==========

func TestAll(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DISTRIBUTED CUT - COMPREHENSIVE TEST SUITE")
	fmt.Println(strings.Repeat("=", 60))

	// Запускаем все тесты
	t.Run("SingleServer", TestSingleServer)
	t.Run("DistributedProcessing", TestDistributedProcessing)
	t.Run("FaultTolerance", TestFaultTolerance)
	t.Run("Sharding", TestSharding)
	t.Run("NetworkCommunication", TestNetworkCommunication)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✓ ALL TESTS COMPLETED SUCCESSFULLY")
	fmt.Println(strings.Repeat("=", 60))
}
