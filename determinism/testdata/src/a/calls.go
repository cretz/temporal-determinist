package a

import (
	"log"
	mathrand "math/rand"
	"time"
)

func CallsTime() { // want CallsTime:"calls non-determistic function time.Now"
	time.Now()
}

func CallsTimeTransitively() { // want CallsTimeTransitively:"calls non-determistic function a.CallsTime"
	CallsTime()
}

func CallsOtherTimeCall() { // want CallsOtherTimeCall:"calls non-determistic function time.Until"
	// Marked non-deterministic because it calls time.Now internally
	time.Until(time.Time{})
}

func Recursion() {
	Recursion()
}

func RecursionWithTimeCall() { // want RecursionWithTimeCall:"calls non-determistic function a.CallsTimeTransitively"
	Recursion()
	CallsTimeTransitively()
}

func MultipleCalls() { // want MultipleCalls:"calls non-determistic function time.Now, calls non-determistic function a.CallsTime"
	time.Now()
	CallsTime()
}

func BadCall() { // want BadCall:"declared non-deterministic"
	Recursion()
}

func IgnoredCall() {
	time.Now()
}

func IgnoredCallTransitive() {
	IgnoredCall()
}

func CallsLog() { // want CallsLog:"calls non-determistic function log.Println"
	log.Println()
}

func CallsMathRandom() { // want CallsMathRandom:"calls non-determistic function math/rand.Int"
	mathrand.Int()
}
