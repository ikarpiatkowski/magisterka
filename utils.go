package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
)

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func fail(err error, format string, args ...any) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), err))
		os.Exit(1)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genString(n int) string {
	b := make([]rune, n)

	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

var keywordVocabulary = []string{
	"mongodb", "postgresql", "elasticsearch", "analiza", "raport", "benchmark",
	"indeks", "transakcja", "replikacja", "skalowalność", "wydajność", "jsonb",
	"optymalizacja", "asynchroniczny", "journaling", "agregacja",
}

var loremIpsumWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "curabitur", "vitae", "hendrerit", "augue",
	"morbi", "ac", "neque", "eu", "nisl", "sollicitudin", "tempor", "sed", "eu", "erat", "phasellus", "sit", "amet", "condimentum",
	"magna", "cras", "euismod", "sapien", "non", "ligula", "auctor", "semper", "quisque", "ut", "dolor", "eget", "nisl",
	"ultricies", "aliquam", "nunc", "molestie", "lacus", "ac", "sodales", "efficitur", "mauris", "magna", "convallis", "nisi",
	"eget", "volutpat", "massa", "sem", "nec", "eros", "nullam", "efficitur", "feugiat", "massa", "sed", "lacinia", "duis",
	"volutpat", "pretium", "elit", "vel", "finibus", "vivamus", "quis", "dignissim", "diam", "aenean", "aliquam", "imperdiet",
	"ante", "in", "convallis", "mauris", "congue", "tempus", "nibh", "quis", "consequat", "vivamus", "laoreet", "porta", "sem",
	"ac", "blandit", "donec", "imperdiet", "lorem", "vel", "facilisis", "ultrices", "risus", "ligula", "posuere", "erat",
	"in", "iaculis", "arcu", "mi", "vel", "augue", "vestibulum", "ante", "ipsum", "primis", "in", "faucibus", "orci", "luctus",
	"et", "ultrices", "posuere", "cubilia", "curae", "suspendisse", "potenti", "nullam", "ac", "tortor", "eu", "felis",
	"tempor", "congue", "sed", "vitae", "arcu", "cras", "non", "dolor", "velit", "maecenas", "tincidunt", "tempus", "turpis",
	"sed", "mollis", "mauris", "in", "est", "bibendum", "tempor", "vivamus", "ultricies", "nisl", "sit", "amet", "finibus",
	"sollicitudin", "lorem", "massa", "aliquet", "libero", "et", "maximus", "nulla", "massa", "a", "ante", "fusce", "hendrerit",
	"risus", "et", "pulvinar", "rutrum", "sapien", "mauris", "vestibulum", "odio", "eu", "efficitur", "erat", "massa", "quis",
	"turpis", "vivamus", "nec", "molestie", "purus", "donec", "eu", "laoreet", "odio", "quisque", "finibus", "luctus", "erat",
	"a", "commodo", "fusce", "eu", "semper", "tellus", "sed", "efficitur", "pharetra", "ipsum",
}

func generateFTSContent() string {
    //
    // MOŻESZ TERAZ BEZPIECZNIE ZMIENIĆ TĘ WARTOŚĆ NA 100, 1000, 10000 itd.
    //
    const textLength = 10000 // Długość tekstu bazowego (w słowach)
    const keywordCount = 3   // Ile rzadkich słów wstrzykniemy

    loremLength := len(loremIpsumWords)
    keywordLength := len(keywordVocabulary)

    // Tworzymy bufor na nasz nowy, długi tekst
    textContent := make([]string, textLength)

    // Wybieramy losowy punkt startowy w naszym słowniku Lorem Ipsum
    // Daje nam to "przesuwane okno", o którym myślałeś.
    start := random(0, loremLength)

    // Wypełniamy bufor 'textLength' słowami, zapętlając słownik Lorem Ipsum
    for i := 0; i < textLength; i++ {
        // Operator modulo (%) sprawia, że gdy dojdziemy do końca
        // słownika, zaczynamy znowu od początku.
        // To pozwala nam generować tekst o dowolnej długości.
        textContent[i] = loremIpsumWords[(start+i)%loremLength]
    }

    // Wstrzyknij 'keywordCount' rzadkich słów w losowe miejsca
    // Ta logika pozostaje bez zmian.
    for i := 0; i < keywordCount; i++ {
        keyword := keywordVocabulary[random(0, keywordLength)]
        position := random(0, textLength)
        textContent[position] = keyword // Nadpisz słowo
    }

    return strings.Join(textContent, " ")
}