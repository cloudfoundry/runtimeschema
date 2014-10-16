package models_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CircusTailorConfig", func() {
	var tailorConfig CircusTailorConfig

	BeforeEach(func() {
		tailorConfig = NewCircusTailorConfig([]string{"ocaml-buildpack", "haskell-buildpack", "bash-buildpack"})
	})

	Context("with defaults", func() {
		It("generates a script for running its tailor", func() {
			commandFlags := []string{
				"-appDir=/app",
				"-buildpackOrder=ocaml-buildpack,haskell-buildpack,bash-buildpack",
				"-buildpacksDir=/tmp/buildpacks",
				"-buildArtifactsCacheDir=/tmp/cache",
				"-outputDroplet=/tmp/droplet",
				"-outputMetadata=/tmp/result.json",
				"-outputBuildArtifactsCache=/tmp/output-cache",
			}

			Ω(tailorConfig.Path()).Should(Equal("/tmp/circus/tailor"))
			Ω(tailorConfig.Args()).Should(ConsistOf(commandFlags))
		})
	})

	Context("with overrides", func() {
		BeforeEach(func() {
			tailorConfig.Set("appDir", "/some/app/dir")
			tailorConfig.Set("outputDroplet", "/some/droplet")
			tailorConfig.Set("outputMetadata", "/some/result/dir")
			tailorConfig.Set("buildpacksDir", "/some/buildpacks/dir")
			tailorConfig.Set("buildArtifactsCacheDir", "/some/cache/dir")
			tailorConfig.Set("outputBuildArtifactsCache", "/some/cache-file")
		})

		It("generates a script for running its tailor", func() {
			commandFlags := []string{
				"-appDir=/some/app/dir",
				"-buildpackOrder=ocaml-buildpack,haskell-buildpack,bash-buildpack",
				"-buildpacksDir=/some/buildpacks/dir",
				"-buildArtifactsCacheDir=/some/cache/dir",
				"-outputDroplet=/some/droplet",
				"-outputMetadata=/some/result/dir",
				"-outputBuildArtifactsCache=/some/cache-file",
			}

			Ω(tailorConfig.Path()).Should(Equal("/tmp/circus/tailor"))
			Ω(tailorConfig.Args()).Should(ConsistOf(commandFlags))
		})
	})

	It("returns the path to the app bits", func() {
		Ω(tailorConfig.AppDir()).To(Equal("/app"))
	})

	It("returns the path to a given buildpack", func() {
		key := "my-buildpack/key/::"
		Ω(tailorConfig.BuildpackPath(key)).To(Equal("/tmp/buildpacks/8b2f72a0702aed614f8b5d8f7f5b431b"))
	})

	It("returns the path to the staging metadata", func() {
		Ω(tailorConfig.OutputMetadata()).To(Equal("/tmp/result.json"))
	})
})
