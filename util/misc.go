package util

var SequentialInts chan int = make(chan int)

func init() {
	i := 0
	go func() {
		for {
			SequentialInts <- i
			i++
		}
	}()
}
