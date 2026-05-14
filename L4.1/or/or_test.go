package or

import (
	"testing"
	"time"
)

func TestOr_NoChannels(t *testing.T) {
	ch := Or()
	if ch != nil {
		t.Errorf("Expected nil channel for zero arguments, got %v", ch)
	}
}

func TestOr_SingleChannel(t *testing.T) {
	original := make(chan interface{})
	result := Or(original)

	// Проверяем, что вернулся тот же канал
	if result != original {
		t.Error("Expected same channel for single argument")
	}
}

func TestOr_TwoChannels_FirstCloses(t *testing.T) {
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})

	result := Or(ch1, ch2)

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(ch1)
	}()

	select {
	case _, ok := <-result:
		if ok {
			t.Error("Expected result channel to be closed")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout: result channel not closed")
	}
}

func TestOr_SecondChannelCloses(t *testing.T) {
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})
	ch3 := make(chan interface{})

	result := Or(ch1, ch2, ch3)

	// Закрываем второй канал через 10ms
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(ch2)
	}()

	select {
	case _, ok := <-result:
		if ok {
			t.Error("Expected result channel to be closed")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Timeout: result channel not closed")
	}
}

func TestOr_WithValues(t *testing.T) {
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})

	result := Or(ch1, ch2)

	// Отправляем значение и закрываем
	go func() {
		ch1 <- "test"
		time.Sleep(10 * time.Millisecond)
		close(ch1)
	}()

	// Канал должен закрыться после ch1
	<-result
	_, ok := <-result
	if ok {
		t.Error("Result channel should be closed")
	}
}
