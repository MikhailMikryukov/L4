package or

func Or(channels ...<-chan interface{}) <-chan interface{} {
	if len(channels) == 0 {
		return nil
	}
	return recursiveMerge(channels...)[0]
}

// Рекурсивно объединяем каналы пока всё не объединится в один
func recursiveMerge(channels ...<-chan interface{}) []<-chan interface{} {
	if len(channels) == 1 {
		return channels
	}

	var mergedChannels []<-chan interface{}
	mergedChannels = make([]<-chan interface{}, 0, len(channels))
	if len(channels)%2 == 1 {
		mergedChannels = append(mergedChannels, channels[len(channels)-1])
	}

	if len(channels)%2 == 0 {
		mergedChannels = make([]<-chan interface{}, 0, len(channels)/2)
	} else {
		mergedChannels = make([]<-chan interface{}, 0, len(channels)/2+1)
		mergedChannels = append(mergedChannels, channels[len(channels)-1])
	}

	for i := 0; i < len(channels)-1; i += 2 {
		c := merge(channels[i], channels[i+1])
		mergedChannels = append(mergedChannels, c)
	}

	return recursiveMerge(mergedChannels...)
}

// Функция объединения двух каналов
func merge(a, b <-chan interface{}) <-chan interface{} {
	c := make(chan interface{})
	go func() {
		for {
			select {
			case v, ok := <-a:
				if ok {
					c <- v
				} else {
					a = nil
				}
			case v, ok := <-b:
				if ok {
					c <- v
				} else {
					b = nil
				}
			}
			if a == nil || b == nil {
				close(c)
				return
			}
		}
	}()
	return c
}
