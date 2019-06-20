// Copyright © 2019 Sergey Nuzhdin ipaq.lw@gmail.com
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

package render

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lwolf/kube-atlas/pkg/fileutil"
	"github.com/lwolf/kube-atlas/pkg/helmexec"
	"github.com/lwolf/kube-atlas/pkg/state"
)

var (
	renderAll bool
	dryRun    bool
)

type releaseType string

const (
	releaseTypeHelm      releaseType = "helm"
	releaseTypeKustomize releaseType = "kustomize"
	releaseTypeRaw       releaseType = "raw"
	releaseTypeNone      releaseType = "none"
)

func releaseContentType(release *state.ReleaseSpec, s *state.ClusterSpec) releaseType {
	rlog := log.With().Str("release", release.Name).Logger()
	chartPath, err := release.GetChartPath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get chart directory")
		return releaseTypeNone
	}
	var fls []os.FileInfo
	if fls, err = ioutil.ReadDir(chartPath); err != nil || len(fls) == 0 {
		rlog.Error().Err(err).Msg("failed to get chart directory content")
		return releaseTypeNone
	}
	var yamlsFound bool
	for _, f := range fls {
		if f.Name() == "Chart.yaml" {
			return releaseTypeHelm
		} else if f.Name() == "kustomization.yaml" {
			return releaseTypeKustomize
		} else if filepath.Ext(f.Name()) == ".yaml" {
			yamlsFound = true
		}
	}
	if yamlsFound {
		return releaseTypeRaw
	}
	return releaseTypeNone
}

func renderHelmChart(release *state.ReleaseSpec, s *state.ClusterSpec) error {
	rlog := log.With().Str("release", release.Name).Logger()
	renderTmp, err := ioutil.TempDir("", "helm-release-")
	if err != nil {
		rlog.Fatal().Err(err).Msg("failed to create temp directory")
	}
	defer func() {
		err := os.RemoveAll(renderTmp)
		if err != nil {
			rlog.Error().Err(err).Msg("failed remove temp directory")
		}
	}()

	helm := helmexec.New(&log.Logger)
	args := []string{"--output-dir", renderTmp, "--name", release.Name}
	if release.Namespace != "" {
		args = append(args, "--namespace", release.Namespace)
	}
	configPath, err := release.GetValuesPath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get values directory")
	}
	for _, configFile := range release.Values {
		fullPath := filepath.Join(configPath, configFile)
		isDir, err := fileutil.IsDir(fullPath)
		if err != nil {
			rlog.Error().Err(err).Msg("failed to check path")
			continue
		}
		if isDir {
			rlog.Error().Err(err).Msg("only files are supported at the moment, skipping directory")
			continue
		}
		if fileutil.Exists(fullPath) {
			args = append(args, "--values", fullPath)
		} else {
			rlog.Error().Str("file", configFile).Msg("values file does not exists, skipping")
		}
	}
	chartPath, err := release.GetChartPath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get chart directory")
		return err
	}
	if err := helm.TemplateRelease(chartPath, args...); err != nil {
		rlog.Error().Err(err).Msg("failed to template release")
		return err
	}
	dstPath, err := release.GetReleasePath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get destination directory")
		return err
	}
	err = os.MkdirAll(dstPath, 0755)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to create directory")
		return err
	}
	rlog.Info().Str("path", dstPath).Msg("destination path for the rendered chart")
	err = os.RemoveAll(dstPath)
	if err != nil {
		rlog.Fatal().Err(err)
	}
	err = os.MkdirAll(dstPath, 0755)
	if err != nil {
		rlog.Fatal().Err(err)
	}
	var fds []os.FileInfo
	// there should be only a single directory after helm template in the temp

	// resultDir := filepath.Join(destTmp, r.Name)
	if fds, err = ioutil.ReadDir(renderTmp); err != nil {
		rlog.Fatal().Err(err).Msg("failed to read directory content")
	}
	chartTmpPath := filepath.Join(renderTmp, fds[0].Name())
	if fds, err = ioutil.ReadDir(chartTmpPath); err != nil {
		rlog.Fatal().Err(err).Msg("failed to read directory content")
	}
	// rlog.Info().Str("path", resultDir).Msg("result path for the rendered chart")
	for _, fd := range fds {
		srcfp := filepath.Join(chartTmpPath, fd.Name())
		dstfp := filepath.Join(dstPath, fd.Name())
		rlog.Debug().Msgf("copy from %s to %s", srcfp, dstfp)
		if fd.IsDir() {
			err = fileutil.CopyDir(srcfp, dstfp)
			if err != nil {
				rlog.Error().Err(err).Msg("error copying dir")
			}
		} else {
			err = fileutil.CopyFile(srcfp, dstfp)
			if err != nil {
				rlog.Error().Err(err).Msg("error copying file")
			}
		}
	}
	return nil
}

func renderKustomize(release *state.ReleaseSpec, s *state.ClusterSpec) error {
	return fmt.Errorf("not implemented error")
}

func renderRaw(release *state.ReleaseSpec, s *state.ClusterSpec) error {
	return fmt.Errorf("not implemented error")
}

func copyManifests(release *state.ReleaseSpec, s *state.ClusterSpec) error {
	rlog := log.With().Str("release", release.Name).Logger()
	manifestsPath, err := release.GetManifestsPath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get manifests path")
		return err
	}
	dstPath, err := release.GetReleasePath(&s.Defaults)
	if err != nil {
		rlog.Error().Err(err).Msg("failed to get destination directory")
		return err
	}
	for _, m := range release.Manifests {
		mlog := rlog.With().Str("manifest", m).Logger()
		mlog.Info().Msg("processing manifest")
		p := filepath.Join(manifestsPath, m)
		if !fileutil.Exists(p) {
			mlog.Error().Err(err).Msg("manifest path does not exist")
			continue
		}
		isDir, err := fileutil.IsDir(p)
		if err != nil {
			mlog.Error().Err(err).Msg("isDir check failed")
			continue
		}
		mlog.Info().Msgf("manifest isDir %v", isDir)
		if isDir {
			err = fileutil.CopyDir(p, dstPath)
			if err != nil {
				mlog.Error().Err(err).Msg("failed to copy directory")
			}
		} else {
			manifestDestPath := filepath.Join(dstPath, fmt.Sprintf("manifest-%s", m))
			rlog.Info().Str("source", p).Str("dst", manifestDestPath).Msg("going to copy raw manifests")
			err = fileutil.CopyFile(p, manifestDestPath)
			if err != nil {
				mlog.Error().Err(err).Msg("failed to copy")
			}
		}
	}
	return nil
}

// renderCmd represents the render command
var CmdRender = &cobra.Command{
	Use:   "render",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := state.LoadSpec()
		if err != nil {
			log.Fatal().Err(err).Msg("unable to unmarshal config")
		}
		err = s.CreateReleaseDirectories()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create destination directories")
		}
		var releases []state.ReleaseSpec
		if renderAll {
			releases = s.Releases
		} else if len(args) > 0 {
			if len(args) > 0 {
				for _, r := range args {
					rl := s.ReleaseByName(r)
					if rl != nil {
						releases = append(releases, *rl)
					}
				}
				if len(releases) == 0 {
					log.Fatal().Strs("names", args).Msg("no releases with these names found in the config")
				}
			}
		} else {
			log.Fatal().Msg("either --all or release name is required")
		}
		err = s.CreateReleaseDirectories()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create target directory structure")
		}
		clusterPath := s.Defaults.GetReleasePath()
		err = os.MkdirAll(clusterPath, 0755)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create release directory")
		}
		for _, r := range releases {
			rlog := log.With().Str("release", r.Name).Logger()
			// validate that chart directory exists and not empty
			_, err := r.GetChartPath(&s.Defaults)
			if err != nil {
				rlog.Error().Err(err).Msg("failed to get chart directory")
				continue
			}
			// process chart directory
			switch releaseContentType(&r, s) {
			case releaseTypeHelm:
				err = renderHelmChart(&r, s)
				if err != nil {
					log.Error().Err(err).Msg("failed to render helm chart")
				}
			case releaseTypeKustomize:
				err = renderKustomize(&r, s)
				if err != nil {
					log.Error().Err(err).Msg("failed to apply kustomization")
				}
			case releaseTypeRaw:
				err = renderRaw(&r, s)
				if err != nil {
					log.Error().Err(err).Msg("failed to copy raw manifests")
				}
			case releaseTypeNone:
			default:
				rlog.Info().Msg("unknown release chart folder content, skipping")
			}
			// process manifests directory
			err = copyManifests(&r, s)
			if err != nil {
				log.Error().Err(err).Msg("failed to copy manifests")
			}
		}
	},
}

func init() {
	CmdRender.Flags().BoolVar(&renderAll, "all", false, "Render all the releases listed in the config")
	CmdRender.Flags().BoolVar(&dryRun, "dry-run", false, "Render to stdout")
}