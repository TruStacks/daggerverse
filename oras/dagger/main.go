// Distribute Artifacts Across OCI Registries With Ease
// https://oras.land/

package main

import (
	"context"
	"fmt"
	"strings"
)

type Oras struct {
	Registry  string
	Username  string
	Password  *Secret
	PlainHTTP bool
	Version   string
	Container *Container
}

func (oras *Oras) login(container *Container) *Container {
	container = container.WithSecretVariable("REGISTRY_PASSWORD", oras.Password)
	container = container.WithEntrypoint([]string{"/bin/sh", "-c"})
	cmd := []string{"oras", "login", "-u", oras.Username, "-p", "$REGISTRY_PASSWORD", oras.Registry}
	if oras.PlainHTTP {
		cmd = append(cmd, "--plain-http")
	}
	return container.
		WithExec([]string{strings.Join(cmd, " ")}).
		WithEntrypoint([]string{"/bin/oras"})
}

func New(
	// OCI registry
	registry string,
	// OCI registry password
	// +optional
	username string,
	// Allow insecure connections to registry without SSL check
	// +optional
	password *Secret,
	// OCI registry username
	// +optional
	plainHttp bool,
	// Oras version
	// +optional
	// +default="1.1.0"
	version string,
) *Oras {
	container := dag.Container().From(fmt.Sprintf("ghcr.io/oras-project/oras:v%s", version))
	oras := &Oras{
		Registry:  registry,
		Username:  username,
		Password:  password,
		PlainHTTP: plainHttp,
		Container: container,
	}
	if oras.Password != nil {
		oras.Container = oras.login(oras.Container)
	}
	return oras

}

// Push an artifact to an oci registry
func (oras *Oras) Push(
	// Artifact source directory
	source *Directory,
	// Artifact name
	name string,
	// Artifact file
	file string,
	// Artifact tag
	tag string,
	// Artifact type
	// +optional
	artifactType string,
) error {
	cmd := []string{"push"}
	if oras.PlainHTTP {
		cmd = append(cmd, "--plain-http")
	}
	if artifactType != "" {
		cmd = append(cmd, "--artifact-type", artifactType)
	}
	cmd = append(cmd, fmt.Sprintf("%s/%s:%s", oras.Registry, name, tag), file)
	_, err := oras.Container.
		WithMountedDirectory("/artifacts", source).
		WithWorkdir("/artifacts").
		WithExec(cmd).
		Sync(context.Background())

	return err
}

// Pull an artifact from an oci registry.
func (oras *Oras) Pull(
	// Artifact source directory
	source *Directory,
	// Artifact name
	name string,
	// Artifact tag
	tag string,
	// Export the directory subpath
	// +optional
	subPath string,
) *Directory {
	cmd := []string{"pull"}
	if oras.PlainHTTP {
		cmd = append(cmd, "--plain-http")
	}
	cmd = append(cmd, fmt.Sprintf("%s/%s:%s", oras.Registry, name, tag), "-o", "/out")
	return oras.Container.
		WithExec(cmd).
		Directory(fmt.Sprintf("/out/%s", subPath))
}
