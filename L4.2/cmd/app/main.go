package main

import (
	"L4.2/internal/cut"
	"L4.2/internal/server"
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// Парсим флаги
	var fieldsStr string
	var delimiter string
	var separated bool
	var serverID int
	var totalServers int
	var port int

	flag.StringVar(&fieldsStr, "f", "", "Fields to print (example: 1,2,3 or 1-5)")
	flag.StringVar(&delimiter, "d", "\t", "Delimiter (default: tab)")
	flag.BoolVar(&separated, "s", false, "Ignore lines without delimiter")
	flag.IntVar(&serverID, "id", 0, "Server ID (0-based)")
	flag.IntVar(&totalServers, "n", 1, "Total number of servers in cluster")
	flag.IntVar(&port, "port", 8000, "Base port for this s (actual port = base + id)")
	flag.Parse()

	// Парсим поля
	fields := cut.ParseFields(fieldsStr)
	if len(fields) == 0 {
		fmt.Fprintf(os.Stderr, "Error: please specify fields with -f flag\n")
		os.Exit(1)
	}

	cutConfig := cut.Config{
		Fields:    fields,
		Delimiter: delimiter,
		Separated: separated,
	}

	// Адрес этого сервера
	myAddress := fmt.Sprintf("localhost:%d", port+serverID)

	// Создаем сервер
	s := server.NewNetworkServer(serverID, totalServers, myAddress, cutConfig)

	// Запускаем сервер
	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start s: %v\n", err)
		os.Exit(1)
	}

	// Формируем адреса пиров
	peerAddresses := make(map[int]string)
	for i := 0; i < totalServers; i++ {
		if i != serverID {
			peerAddresses[i] = fmt.Sprintf("localhost:%d", port+i)
		}
	}

	// Подключаемся к пирам
	if err := s.ConnectToPeers(peerAddresses); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// Даем время на установку соединений
	time.Sleep(2 * time.Second)

	// Читаем входные данные
	var inputData []string

	if flag.NArg() > 0 {
		fileName := flag.Arg(0)
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			inputData = append(inputData, scanner.Text())
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputData = append(inputData, scanner.Text())
		}
	}

	fmt.Printf("=== Distributed Cut System ===\n")
	fmt.Printf("Server ID: %d/%d\n", serverID, totalServers-1)
	fmt.Printf("Address: %s\n", myAddress)
	fmt.Printf("Total lines to process: %d\n", len(inputData))
	fmt.Printf("Fields: %v\n\n", fields)

	// Если всего 1 сервер - просто обрабатываем
	if totalServers == 1 {
		results := cut.ProcessBatch(inputData, cutConfig)
		for _, line := range results {
			fmt.Println(line)
		}
		return
	}

	// Запускаем обработку
	s.ReceiveData(inputData)

	// Запускаем обработку данных в горутине
	go s.ProcessData()

	// Ожидаем кворум
	s.WaitForQuorum()

	// Даем время на отправку commit
	time.Sleep(1 * time.Second)

	// Останавливаем сервер
	s.Stop()
}
