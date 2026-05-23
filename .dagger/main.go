package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/cicd/internal/dagger"
)

const (
	registryHost     = "registry.oscarcorner.com"
	invoiceImageRepo = "registry.oscarcorner.com/invoices"
	ocrImageRepo     = "registry.oscarcorner.com/invoices-ocr"
)

// var releaseTagPattern = regexp.MustCompile(`^v.+`)

type Cicd struct{}

// Ci runs lint, all Go tests, image builds, vulnerability scans, and publish.
func (m *Cicd) Ci(ctx context.Context, source *dagger.Directory, tag string) ([]string, error) {
	if source == nil {
		return nil, fmt.Errorf("source is required (pass --source=.)")
	}
	if strings.TrimSpace(tag) == "" {
		return nil, fmt.Errorf("tag is required")
	}

	if _, err := m.runGolangCILint(ctx, source); err != nil {
		return nil, err
	}
	if _, err := m.runGoTests(ctx, source); err != nil {
		return nil, err
	}

	invoiceImage := source.Directory("invoice-service").DockerBuild()
	ocrImage := source.Directory("ocr-service").DockerBuild()

	if _, err := m.scanWithGrype(ctx, invoiceImage, "invoice-service"); err != nil {
		return nil, err
	}
	if _, err := m.scanWithGrype(ctx, ocrImage, "ocr-service"); err != nil {
		return nil, err
	}

	invoiceRef, err := invoiceImage.Publish(ctx, fmt.Sprintf("%s:%s", invoiceImageRepo, tag))
	if err != nil {
		return nil, fmt.Errorf("publish invoice image: %w", err)
	}
	ocrRef, err := ocrImage.Publish(ctx, fmt.Sprintf("%s:%s", ocrImageRepo, tag))
	if err != nil {
		return nil, fmt.Errorf("publish ocr image: %w", err)
	}

	return []string{invoiceRef, ocrRef}, nil
}

func (m *Cicd) runGolangCILint(ctx context.Context, source *dagger.Directory) (string, error) {
	out, err := dag.Container().
		From("golangci/golangci-lint:v2.12.2-alpine").
		WithMountedDirectory("/src", source.Directory("invoice-service")).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "--timeout=10m", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("golangci-lint failed: %w", err)
	}
	return out, nil
}

func (m *Cicd) runGoTests(ctx context.Context, source *dagger.Directory) (string, error) {
	preparedSource := dag.Container().
		From("node:24-alpine").
		WithMountedDirectory("/src", source.Directory("invoice-service")).
		WithWorkdir("/src").
		WithExec([]string{"corepack", "enable"}).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"mkdir", "-p", "static"}).
		WithExec([]string{"cp", "node_modules/@picocss/pico/css/pico.min.css", "static/"}).
		WithExec([]string{"cp", "node_modules/htmx.org/dist/htmx.min.js", "static/"}).
		Directory("/src")

	out, err := dag.Container().
		From("golang:1.26-bookworm").
		WithMountedDirectory("/src", preparedSource).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1").
		WithExec([]string{"go", "test", "-race", "-count=1", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("go tests failed: %w", err)
	}
	return out, nil
}

func (m *Cicd) scanWithGrype(ctx context.Context, image *dagger.Container, imageName string) (string, error) {
	tarball := image.AsTarball()
	out, err := dag.Container().
		From("anchore/grype:v0.94.0").
		WithMountedFile("/tmp/image.tar", tarball).
		WithExec([]string{"grype", "oci-archive:/tmp/image.tar", "--fail-on", "high"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("grype scan failed for %s: %w", imageName, err)
	}
	return out, nil
}
