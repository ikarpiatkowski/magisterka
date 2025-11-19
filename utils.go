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
    const textLength = 10000
    const keywordCount = 3

    loremLength := len(loremIpsumWords)
    keywordLength := len(keywordVocabulary)

    textContent := make([]string, textLength)

    start := random(0, loremLength)

    for i := 0; i < textLength; i++ {
        textContent[i] = loremIpsumWords[(start+i)%loremLength]
    }

    for i := 0; i < keywordCount; i++ {
        keyword := keywordVocabulary[random(0, keywordLength)]
        position := random(0, textLength)
        textContent[position] = keyword
    }

    return strings.Join(textContent, " ")
}