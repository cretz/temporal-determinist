package a

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"
)

var BadVar time.Time // want BadVar:"declared non-deterministic"

func AccessesStdout() { // want AccessesStdout:"accesses non-determistic var os.Stdout"
	os.Stdout.Write([]byte("Hello"))
}

func AccessesStdoutTransitively() { // want AccessesStdoutTransitively:"calls non-determistic function a.AccessesStdout"
	AccessesStdout()
}

func CallsOtherStdoutCall() { // want CallsOtherStdoutCall:"calls non-determistic function fmt.Println"
	fmt.Println()
}

func AccessesBadVar() { // want AccessesBadVar:"accesses non-determistic var a.BadVar"
	BadVar.Day()
}

func AccessesIgnoredStderr() {
	os.Stderr.Write([]byte("Hello"))
}

func AccessesCryptoRandom() { // want AccessesCryptoRandom:"accesses non-determistic var crypto/rand.Reader"
	rand.Reader.Read(nil)
}

func AccessesCryptoRandomTransitively() { // want AccessesCryptoRandomTransitively:"calls non-determistic function crypto/rand.Read"
	rand.Read(nil)
}
