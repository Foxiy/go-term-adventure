package main

import (
	"./levels"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var encryption_key string

func main() {
	key := []byte(encryption_key)

	print_flag := flag.Bool("print", false, "print loaded levels and exit")
	pretty_print_flag := flag.Bool("no-pretty-print", false,
		"disable option to skip pretty printing")
	detect_level_flag := flag.Bool("detect-level", false,
		"detect a level from a given hash and home directory")
	encrypt_flag := flag.Bool("enc", false, "encrypt a given challenge")
	decrypt_flag := flag.Bool("dec", false, "decrypt a given challenge")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Printf("\n\nNo input file\n\n")
		fmt.Printf("usage: %s path\n", os.Args[0])
		os.Exit(1)
	}
	path := flag.Args()[0]
	challenge_name := levels.BasenameFromPath(path)

	challenge := levels.NewChallenge(challenge_name)
	challenge_text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	if *encrypt_flag {
		encrypted_text, err := levels.Encrypt(key, string(challenge_text))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(encrypted_text)
		os.Exit(0)
	}

	decrypted_text := string(challenge_text)
	if strings.HasSuffix(path, ".enc") {
		decrypted_text, err = levels.Decrypt(key, string(challenge_text))
		if err != nil {
			log.Fatal(err)
		}
	}

	if *decrypt_flag {
		fmt.Println(decrypted_text)
		os.Exit(0)
	}

	challenge.LoadFromString(decrypted_text)

	if *detect_level_flag {
		if len(flag.Args()) < 3 {
			fmt.Printf("usage: %s path hash homedir\n", os.Args[0])
			os.Exit(1)
		}

		level := flag.Args()[1]
		homedir := flag.Args()[2]
		str, i := challenge.IDAndHomedirToLevel(level, homedir)
		if i != -1 {
			fmt.Println("Detected level:", str)
			os.Exit(0)
		} else {
			fmt.Println("Level undetected")
			os.Exit(1)
		}
	}

	if *print_flag {
		challenge.Print()
		os.Exit(0)
	}

	challenge.LoadCfg()

	if challenge.CheckCurrentLevel() {
		challenge.GoToNextLevel()
	}

	if *challenge.LastLevelPrinted != "yes" {
		challenge.PrintCurrentLevel(*pretty_print_flag)
		challenge.SetConfigVal("last_level_printed", "yes")
	}

	challenge.PrintIdentifier()
}
