// Package prepare defines utility service to prepare prior to hyprdynamicmonitors daemon run
package prepare

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fiffeek/hyprdynamicmonitors/internal/config"
	"github.com/fiffeek/hyprdynamicmonitors/internal/utils"
	"github.com/sirupsen/logrus"
)

type Service struct {
	cfg                 *config.Config
	monitorDisableRegex *regexp.Regexp
}

func NewService(cfg *config.Config) *Service {
	monitorDisableRegex := regexp.MustCompile(`.*monitor.*=.*disable.*`)
	return &Service{
		cfg,
		monitorDisableRegex,
	}
}

func (s *Service) TruncateDestination() error {
	file := s.cfg.Get().General.Destination
	_, err := os.Stat(*file)
	if err != nil {
		logrus.WithFields(logrus.Fields{"destination": *file}).Info("file does not exist")
		//nolint:nilerr
		return nil
	}

	contents, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("cant read the %s destination file: %w", *file, err)
	}

	newContents := s.filterDisabledMonitors(string(contents))
	if err := utils.WriteAtomic(*file, []byte(newContents)); err != nil {
		return fmt.Errorf("cant write to %s destination file: %w", *file, err)
	}

	return nil
}

func (s *Service) filterDisabledMonitors(contents string) string {
	if *s.cfg.Get().General.ConfigFormat == config.LuaConfigFormat {
		return s.filterDisabledLuaMonitors(contents)
	}

	lines := strings.Split(contents, "\n")
	var filteredLines []string
	for _, line := range lines {
		if s.monitorDisableRegex.MatchString(line) {
			logrus.Infof("Line %s will be removed", line)
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}

func (s *Service) filterDisabledLuaMonitors(contents string) string {
	lines := strings.Split(contents, "\n")
	filteredLines := []string{}

	inMonitorBlock := false
	blockLines := []string{}
	blockDisabled := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inMonitorBlock && strings.HasPrefix(trimmed, "hl.monitor({") {
			inMonitorBlock = true
			blockLines = []string{line}
			blockDisabled = false
			continue
		}

		if inMonitorBlock {
			blockLines = append(blockLines, line)
			if strings.Contains(trimmed, "disabled = true") {
				blockDisabled = true
			}
			if trimmed == "})" {
				if blockDisabled {
					logrus.Infof("Disabled monitor block will be removed: %s", strings.Join(blockLines, "\n"))
				} else {
					filteredLines = append(filteredLines, blockLines...)
				}
				inMonitorBlock = false
				blockLines = nil
			}
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	if inMonitorBlock {
		filteredLines = append(filteredLines, blockLines...)
	}

	return strings.Join(filteredLines, "\n")
}
