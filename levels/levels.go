package levels

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/rakyll/globalconf"
	"gopkg.in/yaml.v2"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type flagValue struct {
	str string
}

func (f *flagValue) String() string {
	return f.str
}

func (f *flagValue) Set(value string) error {
	f.str = value
	return nil
}

type Level struct {
	Name              string
	PreLevelCmd       string `yaml:"precmd"`
	PostLevelCmd      string `yaml:"postcmd"`
	PostLevelPrintCmd string `yaml:"postprintcmd"`
	Text              string
	TestCmd           string   `yaml:"test"`
	NextLevels        []string `yaml:"next"`
}

func (level *Level) Print(pretty_print_flag bool, print_sleep_time int) {
	terminalized_text := MarkdownToTerminal(level.Text)
	PrintText(terminalized_text, pretty_print_flag, print_sleep_time)
}

func (level *Level) PrintText() {
	terminalized_text := MarkdownToTerminal(level.Text)
	fmt.Println(terminalized_text)
}

type Challenge struct {
	Name             string
	Levels           []Level
	conf             *globalconf.GlobalConf
	CurrentLevel     *string
	LastLevelPrinted *string
}

func NewChallenge(name string) Challenge {
	cfg, err := globalconf.New(name)
	if err != nil {
		log.Fatal(err)
	}

	c := Challenge{
		Name:             name,
		conf:             cfg,
		CurrentLevel:     flag.String("level", LevelToID("", name), "Current Level"),
		LastLevelPrinted: flag.String("last_level_printed", "no", "Last Level Printed"),
	}
	return c
}

func (c *Challenge) SetCurrentLevel(level string) {
	*c.CurrentLevel = LevelToID(level, c.Name)
}

func (c *Challenge) AddLevel(level Level) {
	c.Levels = append(c.Levels, level)
}

// Check whether every level is sane (i.e. it has a non-empty level name).
func (c *Challenge) SanityCheck() {
	last_working_level := "<None> (The first level is likely broken)"
	for _, level := range c.Levels {
		if level.Name != "" {
			last_working_level = level.Name
		} else {
			fmt.Printf("[error] Something is wrong -- a level with an empty name has been found.\n")
			fmt.Printf("[error] The last correct level was '%s'.\n", last_working_level)
			os.Exit(1)
		}
	}
}

func (c *Challenge) CheckCurrentLevel() bool {
	level, index := c.IDToLevel(*c.CurrentLevel)
	passed, next_level := CmdOK(c.Levels[index].TestCmd)
	if next_level == "" {
		return passed
	} else {
		index := c.LevelNameToIndex(next_level)
		if index == -1 {
			return false
		}
		*c.CurrentLevel = LevelToID(level, c.Name)
		return true
	}
}

func (c *Challenge) PrintCurrentLevel(pretty_print_flag bool, print_sleep_time int) {
	_, index := c.IDToLevel(*c.CurrentLevel)
	c.Levels[index].Print(pretty_print_flag, print_sleep_time)
	CmdOK(c.Levels[index].PostLevelPrintCmd)
}

func (c *Challenge) PrintCurrentLevelText() {
	_, index := c.IDToLevel(*c.CurrentLevel)
	c.Levels[index].PrintText()
}

func (c *Challenge) GoToNextLevel() {
	level, index := c.IDToLevel(*c.CurrentLevel)

	CmdOK(c.Levels[index].PostLevelCmd)

	rand.Seed(time.Now().UTC().UnixNano())
	i := rand.Intn(len(c.Levels[index].NextLevels))

	level = c.Levels[index].NextLevels[i]
	index = c.LevelNameToIndex(level)
	CmdOK(c.Levels[index].PreLevelCmd)

	id := LevelToID(level, c.Name)
	c.SetConfigVal("level", id)
	*c.CurrentLevel = id

	c.SetConfigVal("last_level_printed", "no")
	*c.LastLevelPrinted = "no"
}

func (c *Challenge) SetConfigVal(name string, value string) {
	fint := &flagValue{str: value}
	f := &flag.Flag{Name: name, Value: fint}
	c.conf.Set("", f)
}

func (c *Challenge) LoadCfg() {
	c.conf.ParseAll()
}

func (c *Challenge) PrintIdentifier() {
	level, _ := c.IDToLevel(*c.CurrentLevel)
	fmt.Printf("[%s %s]", c.Name, level)
}

func (c *Challenge) PrintLevel() {
	level, _ := c.IDToLevel(*c.CurrentLevel)
	fmt.Printf("%s", level)
}

func (c *Challenge) PrintChallengeName() {
	fmt.Printf("%s", c.Name)
}

func (c *Challenge) LevelNameToIndex(name string) int {
	for i := 0; i < len(c.Levels); i++ {
		if name == c.Levels[i].Name {
			return i
		}
	}
	return -1
}

func (c *Challenge) IDAndHomedirToLevel(id string, homedir string) (string, int) {
	for i := 0; i < len(c.Levels); i++ {
		if id == LevelAndHomedirToID(c.Levels[i].Name, c.Name, homedir) {
			return c.Levels[i].Name, i
		}
	}
	return "", -1
}

func (c *Challenge) IDToLevel(id string) (string, int) {
	for i := 0; i < len(c.Levels); i++ {
		if id == LevelToID(c.Levels[i].Name, c.Name) {
			return c.Levels[i].Name, i
		}
	}
	return "", -1
}

func LevelToID(level string, challenge_name string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return GetMD5Hash(fmt.Sprintf("i%sj%dk%sl", challenge_name, level, usr.HomeDir))
}

func LevelAndHomedirToID(level string, challenge_name string, homedir string) string {
	return GetMD5Hash(fmt.Sprintf("i%sj%dk%sl", challenge_name, level, homedir))
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (c *Challenge) LoadFromString(text string) {
	level_regex, _ := regexp.Compile("(?s)(.*?)\n\n------------+\n")
	for _, part := range level_regex.FindAllStringSubmatch(text, -1) {
		c.AddLevel(buildLevel(part[1]))
	}
	*c.CurrentLevel = LevelToID(c.Levels[0].Name, c.Name)
}

func BasenameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(path)
	return base[:len(base)-len(ext)]
}

func buildLevel(text string) Level {
	parts := strings.Split(text, "\n\n")
	metadata := parts[0]
	clean_text := strings.Join(parts[1:len(parts)], "\n\n")

	level := Level{}

	err := yaml.Unmarshal([]byte(metadata), &level)
	if err != nil {
		log.Fatal(err)
	}
	level.Text = clean_text
	return level
}

func (c *Challenge) Print() {
	fmt.Printf("We have %d levels.\n", len(c.Levels))
	for i := 0; i < len(c.Levels); i++ {
		c.Levels[i].PrintStructured()
	}
}

func (l *Level) PrintStructured() {
	d, err := yaml.Marshal(&l)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%s\n", string(d))
}
