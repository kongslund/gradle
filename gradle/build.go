/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gradle

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libbs"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc := libpak.NewDependencyCache(context.Buildpack)
	dc.Logger = b.Logger

	command := filepath.Join(context.Application.Path, "gradlew")
	if _, err := os.Stat(command); os.IsNotExist(err) {
		dep, err := dr.Resolve("gradle", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		d := NewDistribution(dep, dc, result.Plan)
		d.Logger = b.Logger
		result.Layers = append(result.Layers, d)

		command = filepath.Join(context.Layers.Path, "gradle", "bin", "gradle")
	} else if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to stat %s\n%w", command, err)
	}
	u, err := user.Current()
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	c := libbs.Cache{Path: filepath.Join(u.HomeDir, ".gradle")}
	c.Logger = b.Logger
	result.Layers = append(result.Layers, c)

	arg := libbs.NewArgumentResolver("BP_GRADLE_BUILD_ARGUMENTS", []string{"--no-daemon", "-x", "test", "build"}, b.Logger)

	art := libbs.NewArtifactResolver("BP_GRADLE_BUILT_ARTIFACT", "BP_GRADLE_BUILT_MODULE", filepath.Join("build", "libs", "*.[jw]ar"), b.Logger)
	art.InterestingFileDetector = libbs.JARInterestingFileDetector{}

	a, err := libbs.NewApplication(context.Application.Path, arg, art, c, command, result.Plan)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create application layer\n%w", err)
	}
	a.Logger = b.Logger
	result.Layers = append(result.Layers, a)

	return result, nil
}
