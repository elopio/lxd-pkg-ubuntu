package main

import (
	"strings"

	"github.com/lxc/lxd/shared"

	log "gopkg.in/inconshreveable/log15.v2"
)

/* Patches are one-time actions that are sometimes needed to update
   existing container configuration or move things around on the
   filesystem.

   Those patches are applied at startup time after the database schema
   has been fully updated. Patches can therefore assume a working database.

   At the time the patches are applied, the containers aren't started
   yet and the daemon isn't listening to requests.

   DO NOT use this mechanism for database update. Schema updates must be
   done through the separate schema update mechanism.


   Only append to the patches list, never remove entries and never re-order them.
*/

var patches = []patch{
	patch{name: "invalid_profile_names", run: patchInvalidProfileNames},
}

type patch struct {
	name string
	run  func(name string, d *Daemon) error
}

func (p *patch) apply(d *Daemon) error {
	shared.Debugf("Applying patch: %s", p.name)

	err := p.run(p.name, d)
	if err != nil {
		return err
	}

	err = dbPatchesMarkApplied(d.db, p.name)
	if err != nil {
		return err
	}

	return nil
}

func patchesApplyAll(d *Daemon) error {
	appliedPatches, err := dbPatches(d.db)
	if err != nil {
		return err
	}

	for _, patch := range patches {
		if shared.StringInSlice(patch.name, appliedPatches) {
			continue
		}

		err := patch.apply(d)
		if err != nil {
			return err
		}
	}

	return nil
}

// Patches begin here
func patchInvalidProfileNames(name string, d *Daemon) error {
	profiles, err := dbProfiles(d.db)
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		if strings.Contains(profile, "/") || shared.StringInSlice(profile, []string{".", ".."}) {
			shared.Log.Info("Removing unreachable profile (invalid name)", log.Ctx{"name": profile})
			err := dbProfileDelete(d.db, profile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
