// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/clearlinux/mixer-tools/builder"
	"github.com/clearlinux/mixer-tools/helpers"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type buildCmdFlags struct {
	format        string
	newFormat     string
	increment     bool
	minVersion    int
	clean         bool
	noSigning     bool
	noPublish     bool
	template      string
	skipFullfiles bool
	skipPacks     bool

	numFullfileWorkers int
	numDeltaWorkers    int
	numBundleWorkers   int
}

var buildFlags buildCmdFlags

func setWorkers(b *builder.Builder) {
	var workers int
	workers = buildFlags.numFullfileWorkers
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	b.NumFullfileWorkers = workers
	workers = buildFlags.numDeltaWorkers
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	b.NumDeltaWorkers = workers
	workers = buildFlags.numBundleWorkers
	if workers < 1 {
		workers = runtime.NumCPU()
	}
	b.NumBundleWorkers = workers
}

// buildCmd represents the base build command when called without any subcommands
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build various pieces of OS content",
}

func buildBundles(builder *builder.Builder, signflag, cleanFlag bool) error {
	// Create the signing and validation key/cert
	if _, err := os.Stat(builder.Config.Builder.Cert); os.IsNotExist(err) {
		fmt.Println("Generating certificate for signature validation...")
		privkey, err := helpers.CreateKeyPair()
		if err != nil {
			return errors.Wrap(err, "Error generating OpenSSL keypair")
		}
		template := helpers.CreateCertTemplate()

		err = builder.BuildBundles(template, privkey, signflag, cleanFlag)
		if err != nil {
			return errors.Wrap(err, "Error building bundles")
		}
	} else {
		err := builder.BuildBundles(nil, nil, true, cleanFlag)
		if err != nil {
			return errors.Wrap(err, "Error building bundles")
		}
	}
	return nil
}

var buildBundlesCmd = &cobra.Command{
	Use:     "bundles",
	Aliases: []string{"chroots"},
	Short:   "Build the bundles for your mix",
	Long:    `Build the bundles for your mix`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}
		setWorkers(b)
		err = buildBundles(b, buildFlags.noSigning, buildFlags.clean)
		if err != nil {
			fail(err)
		}
	},
}

var buildUpstreamFormatCmd = &cobra.Command{
	Use:    "upstream-format",
	Short:  "Use to create the necessary builds to cross an upstream format",
	Long:   `Use to create the necessary builds to cross an upstream format`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}

		// Don't print any more warnings about being behind formats when we loop
		silent := true
		bumpNeeded := true

		for bumpNeeded {
			cmdStr := fmt.Sprintf("mixer build format-bump old --new-format %s --native", buildFlags.newFormat)
			cmdToRun := strings.Split(cmdStr, " ")
			if err = helpers.RunCommand(cmdToRun[0], cmdToRun[1:]...); err != nil {
				fail(err)
			}

			// Set the upstream version to the previous format's latest version
			b.UpstreamVer = strconv.FormatUint(uint64(b.UpstreamVerUint32), 10)
			vFile := filepath.Join(b.Config.Builder.VersionPath, b.UpstreamVerFile)
			if err = ioutil.WriteFile(vFile, []byte(b.UpstreamVer), 0644); err != nil {
				fail(err)
			}
			cmdStr = fmt.Sprintf("mixer build format-bump new --new-format %s --native", buildFlags.newFormat)
			cmdToRun = strings.Split(cmdStr, " ")
			if err = helpers.RunCommand(cmdToRun[0], cmdToRun[1:]...); err != nil {
				fail(err)
			}
			// Set the upstream version back to what the user originally tried to build
			if err = b.UnstageMixFromBump(); err != nil {
				fail(err)
			}
			bumpNeeded, err = b.CheckBumpNeeded(silent)
			if err != nil {
				fail(err)
			}
		}
		newFormatVer, err := strconv.Atoi(b.MixVer)
		if err != nil {
			failf("Couldn't get new format version")
		}
		newFormatVer += 10
		if err = b.UpdateMixVer(newFormatVer); err != nil {
			failf("Couldn't update Mix Version")
		}
	},
}

var buildFormatBumpCmd = &cobra.Command{
	Use:    "format-bump",
	Short:  "Used to create a downstream format bump",
	Long:   `Used to create a downstream format bump`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if buildFlags.newFormat == "" {
			fail(errors.New("Please supply the next format version with --new-format"))
		}

		cmdStr := fmt.Sprintf("mixer build format-bump old --new-format %s --native", buildFlags.newFormat)
		cmdToRun := strings.Split(cmdStr, " ")
		if output, err := helpers.RunCommandOutputEnv(cmdToRun[0], cmdToRun[1:], []string{}); err != nil {
			failf("%s: %s", output, err)
		}
		cmdStr = fmt.Sprintf("mixer build format-bump new --new-format %s --native", buildFlags.newFormat)
		cmdToRun = strings.Split(cmdStr, " ")
		if output, err := helpers.RunCommandOutputEnv(cmdToRun[0], cmdToRun[1:], []string{}); err != nil {
			failf("%s: %s", output, err)
		}
	},
}

// This is the last build in the original format. At this point add ONLY the
// content relevant to the format bump to the mash to be used. Relevant content
// should be the only change.
//
// mixer will create manifests and update content based on the format it is
// building for. The format is set in the mixer.state file.
var buildFormatOldCmd = &cobra.Command{
	Use:   "old",
	Short: "Build the +10 version in the old format for the format bump",
	Long:  `Build the +10 version in the old format for the format bump`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}

		setWorkers(b)

		lastVer, err := b.GetLastBuildVersion()
		if err != nil {
			fail(err)
		}
		originalVer, err := strconv.Atoi(lastVer)
		if err != nil {
			fail(err)
		}

		oldFormatVer := originalVer + 10
		newFormatVer := originalVer + 20
		// Update mixer to build version +20, to build the bundles with +20 inside not +10
		if err = b.UpdateMixVer(newFormatVer); err != nil {
			failf("Couldn't update Mix Version")
		}

		// Change the format internally just for building bundles so the right
		// /usr/share/defaults/swupd/format is inserted; preserve old format
		oldFormat := b.State.Mix.Format
		b.State.Mix.Format = buildFlags.newFormat

		// Build bundles normally. At this point the bundles to be deleted should still
		// be part of the mixbundles list and the groups.ini
		if err = buildBundles(b, buildFlags.noSigning, buildFlags.clean); err != nil {
			fail(err)
		}

		// Remove deleted bundles and replace with empty dirs for update to mark as deleted
		if err = b.ModifyBundles(b.ReplaceInfoBundles); err != nil {
			fail(err)
		}

		// Link the +20 bundles to the +10 so we are building the update with the same
		// underlying content. The only things that might change are the manifests
		// (potentially the pack and full-file formats as well, though this is very
		// rare).
		if err = b.UpdateMixVer(oldFormatVer); err != nil {
			failf("Couldn't update Mix Version")
		}
		source := filepath.Join(b.Config.Builder.ServerStateDir, "image", strconv.Itoa(newFormatVer))
		dest := filepath.Join(b.Config.Builder.ServerStateDir, "image", strconv.Itoa(oldFormatVer))
		fmt.Println(" Copying +20 bundles to +10 bundles")
		if err = helpers.RunCommandSilent("cp", "-al", source, dest); err != nil {
			failf("Failed to copy +10 bundles to +20: %s\n", err)
		}

		// Set the format back to old for the actual build update
		b.State.Mix.Format = oldFormat

		// Build the update content for the +10 build
		params := builder.UpdateParameters{
			MinVersion:    buildFlags.minVersion,
			Format:        b.State.Mix.Format,
			Publish:       !buildFlags.noPublish,
			SkipSigning:   buildFlags.noSigning,
			SkipFullfiles: buildFlags.skipFullfiles,
			SkipPacks:     buildFlags.skipPacks,
		}
		if err = b.BuildUpdate(params); err != nil {
			failf("Couldn't build update: %s", err)
		}

		// Write the +0 version to LAST_VER so that we reference in both +10 and +20 manifests as the 'previous:'
		if err = ioutil.WriteFile(filepath.Join(b.Config.Builder.ServerStateDir, "image", "LAST_VER"), []byte(lastVer), 0644); err != nil {
			failf("Couldn't update LAST_VER file: %s", err)
		}
	},
}

// This is the first build in the new format. The content is the same as the +10
// but the manifests might be created differently if a new manifest template is
// defined for the new format.
var buildFormatNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Build the +20 version in the new format for the format bump",
	Long:  `Build the +20 version in the new format for the format bump`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}

		// Set mixversion to +20 build now, we only need to build update since we
		// already have the bundles created
		ver, err := strconv.Atoi(b.MixVer)
		if err != nil {
			fail(err)
		}
		if err = b.UpdateMixVer(ver + 10); err != nil {
			failf("Couldn't update Mix Version")
		}

		// Set format to format+1 for future builds
		if err = b.UpdateFormatVersion(buildFlags.newFormat); err != nil {
			fail(err)
		}

		setWorkers(b)

		if err = b.ModifyBundles(b.RemoveBundlesGroupINI); err != nil {
			fail(err)
		}

		minver, err := strconv.Atoi(b.MixVer)
		if err != nil {
			fail(err)
		}

		// Build the +20 update so we don't have to switch tooling in between
		params := builder.UpdateParameters{
			MinVersion:    minver,
			Format:        buildFlags.newFormat,
			Publish:       !buildFlags.noPublish,
			SkipSigning:   buildFlags.noSigning,
			SkipFullfiles: buildFlags.skipFullfiles,
			SkipPacks:     buildFlags.skipPacks,
		}
		err = b.BuildUpdate(params)
		if err != nil {
			failf("Couldn't build update: %s", err)
		}
	},
}

var buildUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Build the update content for your mix",
	Long:  `Build the update content for your mix`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}
		setWorkers(b)
		params := builder.UpdateParameters{
			MinVersion:    buildFlags.minVersion,
			Format:        buildFlags.format,
			Publish:       !buildFlags.noPublish,
			SkipSigning:   buildFlags.noSigning,
			SkipFullfiles: buildFlags.skipFullfiles,
			SkipPacks:     buildFlags.skipPacks,
		}
		err = b.BuildUpdate(params)
		if err != nil {
			failf("Couldn't build update: %s", err)
		}

		if buildFlags.increment {
			ver, err := strconv.Atoi(b.MixVer)
			if err != nil {
				fail(err)
			}
			if err = b.UpdateMixVer(ver + 10); err != nil {
				failf("Couldn't update Mix Version")
			}
		}
	},
}

var buildAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Build all content for mix with default options",
	Long:  `Build all content for mix with default options`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}
		setWorkers(b)
		rpms, err := helpers.ListVisibleFiles(b.Config.Mixer.LocalRPMDir)
		if err == nil {
			err = b.AddRPMList(rpms)
			if err != nil {
				failf("Couldn't add the RPMs: %s", err)
			}
		}
		err = buildBundles(b, buildFlags.noSigning, buildFlags.clean)
		if err != nil {
			failf("Couldn't build bundles: %s", err)
		}
		params := builder.UpdateParameters{
			MinVersion:    buildFlags.minVersion,
			Format:        buildFlags.format,
			Publish:       !buildFlags.noPublish,
			SkipSigning:   buildFlags.noSigning,
			SkipFullfiles: buildFlags.skipFullfiles,
			SkipPacks:     buildFlags.skipPacks,
		}
		err = b.BuildUpdate(params)
		if err != nil {
			failf("Couldn't build update: %s", err)
		}

		if buildFlags.increment {
			ver, err := strconv.Atoi(b.MixVer)
			if err != nil {
				fail(err)
			}
			if err = b.UpdateMixVer(ver + 10); err != nil {
				failf("Couldn't update Mix Version")
			}
		}
	},
}

var buildImageCmd = &cobra.Command{
	Use:   "image",
	Short: "Build an image from the mix content",
	Long:  `Build an image from the mix content`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := builder.NewFromConfig(configFile)
		if err != nil {
			fail(err)
		}
		setWorkers(b)
		err = b.BuildImage(buildFlags.format, buildFlags.template)
		if err != nil {
			failf("Couldn't build image: %s", err)
		}
	},
}

var buildDeltaPacksCmd = &cobra.Command{
	Use:   "delta-packs",
	Short: "Build packs used to optimize update between versions",
	Long: `Build packs used to optimize update between versions

When a swupd client updates a bundle, it looks for a pack file from
its current version to the new version. If not available, the client
will download the individual files necessary for the update. If a
bundle haven't changed between two versions, no pack need to be
generated.

To generate the packs to optimize update from VER to the current mix
version use

    mixer build delta-packs --from VER

Alternatively, to generate packs for a set of NUM previous versions,
each one to the current mix version, instead of --from use

    mixer build delta-packs --previous-versions NUM

To change the target version (by default the current version), use the
flag --to. The target version must be larger than the --from version.

`,
	RunE: runBuildDeltaPacks,
}

var buildDeltaPacksFlags struct {
	previousVersions uint32
	from             uint32
	to               uint32
	report           bool
}

func runBuildDeltaPacks(cmd *cobra.Command, args []string) error {
	fromChanged := cmd.Flags().Changed("from")
	prevChanged := cmd.Flags().Changed("previous-versions")
	if fromChanged == prevChanged {
		return errors.Errorf("either --from or --previous-versions must be set, but not both")
	}

	b, err := builder.NewFromConfig(configFile)
	if err != nil {
		fail(err)
	}
	setWorkers(b)
	if fromChanged {
		err = b.BuildDeltaPacks(buildDeltaPacksFlags.from, buildDeltaPacksFlags.to, buildDeltaPacksFlags.report)
	} else {
		err = b.BuildDeltaPacksPreviousVersions(buildDeltaPacksFlags.previousVersions, buildDeltaPacksFlags.to, buildDeltaPacksFlags.report)
	}
	if err != nil {
		fail(err)
	}
	return nil
}

func setUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&buildFlags.format, "format", "", "Supply format to use")
	cmd.Flags().BoolVar(&buildFlags.increment, "increment", false, "Automatically increment the mixversion post build")
	cmd.Flags().IntVar(&buildFlags.minVersion, "min-version", 0, "Supply minversion to build update with")
	cmd.Flags().BoolVar(&buildFlags.noSigning, "no-signing", false, "Do not generate a certificate and do not sign the Manifest.MoM")
	cmd.Flags().BoolVar(&buildFlags.noPublish, "no-publish", false, "Do not update the latest version after update")
	cmd.Flags().BoolVar(&buildFlags.skipFullfiles, "skip-fullfiles", false, "Do not generate fullfiles")
	cmd.Flags().BoolVar(&buildFlags.skipPacks, "skip-packs", false, "Do not generate zero packs")

	var unusedStringFlag string
	cmd.Flags().StringVar(&unusedStringFlag, "prefix", "", "Supply prefix for where the swupd binaries live")
	_ = cmd.Flags().MarkHidden("prefix")
	_ = cmd.Flags().MarkDeprecated("prefix", "this flag is ignored by the update builder")
	var unusedBoolFlag bool
	cmd.Flags().BoolVar(&unusedBoolFlag, "keep-chroots", false, "Keep individual chroots created and not just consolidated 'full'")
	_ = cmd.Flags().MarkHidden("keep-chroots")
	_ = cmd.Flags().MarkDeprecated("keep-chroots", "this flag is ignored by the update builder")
}

var buildCmds = []*cobra.Command{
	buildBundlesCmd,
	buildUpdateCmd,
	buildAllCmd,
	buildImageCmd,
	buildDeltaPacksCmd,
	buildUpstreamFormatCmd,
	buildFormatBumpCmd,
}

func init() {
	for _, cmd := range buildCmds {
		buildCmd.AddCommand(cmd)
		cmd.Flags().StringVarP(&configFile, "config", "c", "", "Builder config to use")
	}

	buildFormatBumpCmd.AddCommand(buildFormatNewCmd)
	buildFormatBumpCmd.AddCommand(buildFormatOldCmd)
	buildFormatBumpCmd.Flags().StringVar(&buildFlags.newFormat, "new-format", "", "Supply the next format version to build mixes in")
	buildFormatOldCmd.Flags().StringVar(&buildFlags.newFormat, "new-format", "", "Supply the next format version to build mixes in")
	buildFormatNewCmd.Flags().StringVar(&buildFlags.newFormat, "new-format", "", "Supply the next format version to build mixes in")
	buildUpstreamFormatCmd.Flags().StringVar(&buildFlags.newFormat, "new-format", "", "Supply the next format version to build mixes in")

	buildCmd.PersistentFlags().IntVar(&buildFlags.numFullfileWorkers, "fullfile-workers", 0, "Number of parallel workers when creating fullfiles, 0 means number of CPUs")
	buildCmd.PersistentFlags().IntVar(&buildFlags.numDeltaWorkers, "delta-workers", 0, "Number of parallel workers when creating deltas, 0 means number of CPUs")
	buildCmd.PersistentFlags().IntVar(&buildFlags.numBundleWorkers, "bundle-workers", 0, "Number of parallel workers when building bundles, 0 means number of CPUs")

	RootCmd.AddCommand(buildCmd)

	buildBundlesCmd.Flags().BoolVar(&buildFlags.clean, "clean", false, "Wipe the /image and /www dirs if they exist")
	buildBundlesCmd.Flags().BoolVar(&buildFlags.noSigning, "no-signing", false, "Do not generate a certificate to sign the Manifest.MoM")

	unusedBoolFlag := false
	buildBundlesCmd.Flags().BoolVar(&unusedBoolFlag, "new-chroots", false, "")
	_ = buildBundlesCmd.Flags().MarkHidden("new-chroots")
	_ = buildBundlesCmd.Flags().MarkDeprecated("new-chroots", "new functionality is now the standard behavior, this flag is obsolete and no longer used")

	buildImageCmd.Flags().StringVar(&buildFlags.format, "format", "", "Supply the format used for the Mix")
	buildImageCmd.Flags().StringVar(&buildFlags.template, "template", "", "Path to template file to use")

	buildDeltaPacksCmd.Flags().Uint32Var(&buildDeltaPacksFlags.from, "from", 0, "Generate packs from a specific version")
	buildDeltaPacksCmd.Flags().Uint32Var(&buildDeltaPacksFlags.previousVersions, "previous-versions", 0, "Generate packs for multiple previous versions")
	buildDeltaPacksCmd.Flags().Uint32Var(&buildDeltaPacksFlags.to, "to", 0, "Generate packs targeting a specific version")
	buildDeltaPacksCmd.Flags().BoolVar(&buildDeltaPacksFlags.report, "report", false, "Report reason each file in to manifest was packed or not")

	setUpdateFlags(buildUpdateCmd)
	setUpdateFlags(buildAllCmd)
	setUpdateFlags(buildFormatNewCmd)
	setUpdateFlags(buildFormatOldCmd)

	externalDeps[buildBundlesCmd] = []string{
		"rpm",
		"dnf",
	}
	externalDeps[buildUpdateCmd] = []string{
		"openssl",
		"xz",
	}
	externalDeps[buildImageCmd] = []string{
		"ister.py",
	}
	externalDeps[buildAllCmd] = append(
		externalDeps[buildBundlesCmd],
		append(externalDeps[buildUpdateCmd],
			externalDeps[buildImageCmd]...)...)
}
