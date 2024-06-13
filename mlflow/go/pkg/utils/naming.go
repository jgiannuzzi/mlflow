package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

var generatorNouns = []string{
	"ant", "ape", "asp", "auk", "bass", "bat", "bear", "bee", "bird", "boar",
	"bug", "calf", "carp", "cat", "chimp", "cod", "colt", "conch", "cow",
	"crab", "crane", "croc", "crow", "cub", "deer", "doe", "dog", "dolphin",
	"donkey", "dove", "duck", "eel", "elk", "fawn", "finch", "fish", "flea",
	"fly", "foal", "fowl", "fox", "frog", "gnat", "gnu", "goat", "goose",
	"grouse", "grub", "gull", "hare", "hawk", "hen", "hog", "horse", "hound",
	"jay", "kit", "kite", "koi", "lamb", "lark", "loon", "lynx", "mare",
	"midge", "mink", "mole", "moose", "moth", "mouse", "mule", "newt", "owl",
	"ox", "panda", "penguin", "perch", "pig", "pug", "quail", "ram", "rat",
	"ray", "robin", "roo", "rook", "seal", "shad", "shark", "sheep", "shoat",
	"shrew", "shrike", "shrimp", "skink", "skunk", "sloth", "slug", "smelt",
	"snail", "snake", "snipe", "sow", "sponge", "squid", "squirrel", "stag",
	"steed", "stoat", "stork", "swan", "tern", "toad", "trout", "turtle",
	"vole", "wasp", "whale", "wolf", "worm", "wren", "yak", "zebra",
}

var generatorPredicates = []string{
	"abundant", "able", "abrasive", "adorable", "adaptable", "adventurous",
	"aged", "agreeable", "ambitious", "amazing", "amusing", "angry",
	"auspicious", "awesome", "bald", "beautiful", "bemused", "bedecked", "big",
	"bittersweet", "blushing", "bold", "bouncy", "brawny", "bright", "burly",
	"bustling", "calm", "capable", "carefree", "capricious", "caring",
	"casual", "charming", "chill", "classy", "clean", "clumsy", "colorful",
	"crawling", "dapper", "debonair", "dashing", "defiant", "delicate",
	"delightful", "dazzling", "efficient", "enchanting", "entertaining",
	"enthused", "exultant", "fearless", "flawless", "fortunate", "fun",
	"funny", "gaudy", "gentle", "gifted", "glamorous", "grandiose",
	"gregarious", "handsome", "hilarious", "honorable", "illustrious",
	"incongruous", "indecisive", "industrious", "intelligent", "inquisitive",
	"intrigued", "invincible", "judicious", "kindly", "languid", "learned",
	"legendary", "likeable", "loud", "luminous", "luxuriant", "lyrical",
	"magnificent", "marvelous", "masked", "melodic", "merciful", "mercurial",
	"monumental", "mysterious", "nebulous", "nervous", "nimble", "nosy",
	"omniscient", "orderly", "overjoyed", "peaceful", "painted", "persistent",
	"placid", "polite", "popular", "powerful", "puzzled", "rambunctious",
	"rare", "rebellious", "respected", "resilient", "righteous", "receptive",
	"redolent", "resilient", "rogue", "rumbling", "salty", "sassy", "secretive",
	"selective", "sedate", "serious", "shivering", "skillful", "sincere",
	"skittish", "silent", "smiling",
}

func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}

	return int(n.Int64())
}

const BaseTen = 10

func generateString(sep string, integerScale int) string {
	predicate := strings.ToLower(generatorPredicates[randomInt(len(generatorPredicates))])
	noun := strings.ToLower(generatorNouns[randomInt(len(generatorNouns))])
	num := randomInt(intPow(BaseTen, integerScale))

	return fmt.Sprintf("%s%s%s%s%d", predicate, sep, noun, sep, num)
}

func intPow(base, exp int) int {
	result := 1

	for exp != 0 {
		if exp%2 != 0 {
			result *= base
		}

		exp /= 2
		base *= base
	}

	return result
}

func GenerateRandomName(sep string, integerScale, maxLength int) string {
	for i := 0; i < 10; i++ {
		name := generateString(sep, integerScale)
		if len(name) <= maxLength {
			return name
		}
	}

	return generateString(sep, integerScale)[:maxLength]
}
