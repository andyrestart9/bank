package main

import "fmt"

// 只需要「說話」能力，所以介面就留一個方法
type Speaker interface {
	Speak() string
}

// 高階函式：跟誰說話都可以，只要能 Speak
func Announce(s Speaker) {
	fmt.Println(s.Speak())
}

// --------- 不同實作，各自滿足 Speaker ---------

type Human struct{ Name string }

func (h Human) Speak() string { return "Hi, I'm " + h.Name }

type Robot struct{ ID int }

func (r Robot) Speak() string { return fmt.Sprintf("Beep! I am robot #%d", r.ID) }

func main() {
	Announce(Human{"Andy"})   // Hi, I'm Andy
	Announce(Robot{42})       // Beep! I am robot #42
}