package model

import (
	"bufio"
	"encoding/hex"
	"hash/fnv"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gobuffalo/packr/v2"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const (
	// contextsize is the amount of lines (before and after)
	// the violating line that will be added to the snippet
	contextSize = 2
)

type RuleSet struct {
	ApiVersion        string `yaml:"apiVersion"`
	Checksum          string
	ReadAt            time.Time
	Rules             []Rule   `yaml:"rules"`
	BlackList         []string `yaml:"black_list"`
	BlackListCompiled []*regexp.Regexp
}

type Rule struct {
	Definition  string  `yaml:"definition"`
	Description string  `yaml:"description,omitempty"`
	Category    string  `yaml:"category,omitempty"`
	Weight      float32 `yaml:"weight"`
	Compiled    *regexp.Regexp
}

func (r *RuleSet) ParseConfig(file string) {
	var data []byte
	var err error

	if file == "" {
		box := packr.New("rules", "../../resources")
		data, err = box.Find("rules.yaml")
		if err != nil {
			log.Fatal().
				Str("path", file).
				Err(err).
				Msg("Failed to read static binary definition")
		}
	} else {
		data, err = ioutil.ReadFile(file)
		if err != nil {
			log.Fatal().
				Str("path", file).
				Err(err).
				Msg("Failed to read rules definition @")
		}
	}
	if err := yaml.Unmarshal(data, &r); err != nil {
		log.Fatal().
			Str("path", file).
			Err(err).
			Msg("Failed to unmarshal yaml @")
	}

	r.Checksum = hex.EncodeToString(fnv.New32().Sum(data))[:10]
	r.ReadAt = time.Now()

	for idx, rule := range r.Rules {
		r.Rules[idx].Compiled = regexp.MustCompile(rule.Definition)
	}

	for _, bl := range r.BlackList {
		r.BlackListCompiled = append(r.BlackListCompiled, regexp.MustCompile(bl))
	}
}

func (r *RuleSet) ParsePatch(patch *object.Patch, commit *object.Commit, repo *Repo, leakChan chan GitLeak) {
	for _, filePatch := range patch.FilePatches() {
		if filePatch.IsBinary() {
			break
		}
		_, to := filePatch.Files()
		if to == nil {
			continue
		}
		for _, blacklist := range r.BlackListCompiled {
			if blacklist.MatchString(to.Path()) {
				break
			}
		}

		for _, chunk := range filePatch.Chunks() {
			if chunk.Type() == diff.Add {
				lines := strings.Split(strings.Replace(chunk.Content(), "\r\n", "\n", -1), "\n")
				for idx, line := range lines {
					for _, rule := range r.Rules {
						if rule.Compiled.MatchString(line) {
							start := idx - contextSize
							end := idx + contextSize
							if start < 0 {
								start = 0
							}
							if end >= len(lines) {
								end = len(lines) - 1
							}
							disc := GitLeak{
								Line:     idx,
								Affected: idx - start,
								File:     to.Path(),
								Author:   commit.Author.Name,
								When:     commit.Author.When,
								Commit:   to.Hash().String(),
								Repo:     repo,
								Rule:     &rule,
							}
							disc.Snippet = make([]string, len(lines[start:end]))
							copy(disc.Snippet, lines[start:end])
							leakChan <- disc
							break
						}
					}
				}
			}
		}
	}
}

func (r *RuleSet) ParseFile(file string, leakChan chan FileLeak) {
	fd, err := os.Open(file)
	if err != nil {
		log.Error().
			Str("file", file).
			Err(err).
			Msg("Failed to read")
	}
	scanner := bufio.NewScanner(fd)
	defer fd.Close()

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if !utf8.ValidString(line) {
			continue
		}
		for _, rule := range r.Rules {
			if rule.Compiled.MatchString(line) {
				stat, err := fd.Stat()
				if err != nil {
					log.Error().
						Str("file", file).
						Msg("Failed to fetch stat from")
					continue
				}
				leakChan <- FileLeak{
					File:     file,
					Line:     lineNum,
					Affected: line,
					Rule:     &rule,
					Size:     stat.Size(),
				}
				break
			}
		}
	}
}
