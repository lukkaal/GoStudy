package main

import (
	"log"
)

func Funlv1Safe() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Funlv1 catch panic , err is ", err)
		}
		log.Println("Funlv1 exit ...")
	}()
	log.Println("Funlv1 begin")
	panic("sorry, Funlv1 panic")
	log.Println("Funlv1 end")
}

func Funlv2Safe() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Funlv2 catch panic , err is ", err)
		}
		log.Println("Funlv2 exit ...")
	}()
	log.Println("Funlv2 begin")
	Funlv1Safe()
	panic("sorry, Funlv1 panic")
	log.Println("Funlv2 end")
}

func main() {
	Funlv2Safe()
}
