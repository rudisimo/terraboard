package state

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/camptocamp/terraboard/config"
	"github.com/camptocamp/terraboard/internal/terraform/states/statefile"
	log "github.com/sirupsen/logrus"
)

type (
	LocalCollector = func(string, fs.WalkDirFunc) error
	LocalMatcher   = func(string, string) (bool, error)
	Local          struct {
		path      string
		pattern   string
		walkDirFn LocalCollector
		matchFn   LocalMatcher
	}
)

// NewLocal creates a Local backend object
func NewLocal(c config.LocalConfig) (*Local, error) {
	var instance *Local

	if c.StatePath == "" {
		return nil, fmt.Errorf("state path cannot be empty")
	}

	if c.StateFile == "" {
		return nil, fmt.Errorf("state file cannot be empty")
	}

	instance = &Local{
		path:      c.StatePath,
		pattern:   c.StateFile,
		walkDirFn: filepath.WalkDir,
		matchFn:   filepath.Match,
	}

	return instance, nil
}

// NewLocalCollection returns a slice of all instantiated Local backend objects configured by the user
func NewLocalCollection(c *config.Config) ([]*Local, error) {
	var instances []*Local

	for _, config := range c.Local {
		instance, err := NewLocal(config)
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetLocks returns a map of locks (empty for Local)
func (o *Local) GetLocks() (map[string]LockInfo, error) {
	locks := make(map[string]LockInfo)
	return locks, nil
}

// GetStates returns a slice of all State files found in the local filesystem
func (o *Local) GetStates() ([]string, error) {
	var states []string
	log.WithFields(log.Fields{
		"path":    o.path,
		"pattern": o.pattern,
	}).Debug("Listing states from Local")

	err := o.walkDirFn(o.path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"path":  path,
			}).Error("Error retrieving state for Local backend")
			return err
		}
		if !info.IsDir() {
			if matched, err := o.matchFn(o.pattern, filepath.Base(path)); err != nil {
				log.WithFields(log.Fields{
					"error":   err,
					"path":    path,
					"pattern": o.pattern,
				}).Error("Error matching state for Local backend")
				return err
			} else if matched {
				states = append(states, path)
			}
		}
		return nil
	})

	log.WithFields(log.Fields{
		"path":    o.path,
		"pattern": o.pattern,
	}).Debugf("Found %d states in Local", len(states))

	return states, err
}

// GetState retrieves a single State file from the local filesystem
func (o *Local) GetState(state string, version string) (*statefile.File, error) {
	log.WithFields(log.Fields{
		"path":       state,
		"version_id": version,
	}).Info("Retrieving state from Local")

	// Open the state file
	fh, err := os.Open(state)
	if err != nil {
		return nil, fmt.Errorf("unable to read the statefile %s", state)
	}
	defer fh.Close()

	// Parse the state file
	sf, _ := statefile.Read(bufio.NewReader(fh))
	if sf == nil {
		return nil, fmt.Errorf("unable to parse the statefile version %s", version)
	}

	return sf, nil
}

// GetVersions returns a slice of Version objects
func (o *Local) GetVersions(state string) ([]Version, error) {
	var versions []Version

	info, err := os.Stat(state)
	if err != nil {
		return nil, fmt.Errorf("unable to read the statefile %s", state)
	}

	versions = append(versions, Version{
		ID:           state,
		LastModified: info.ModTime(),
	})

	return versions, nil
}
