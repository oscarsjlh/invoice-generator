package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"dagger/cicd/internal/dagger"
)

const (
	registryHost     = "registry.oscarcorner.com"
	invoiceImageRepo = "registry.oscarcorner.com/invoices"
	ocrImageRepo     = "registry.oscarcorner.com/invoices-ocr"
)

var releaseTagPattern = regexp.MustCompile(`^v.+`)

type Cicd struct{}

// TestInvoiceService runs the invoice-service test suite.
func (m *Cicd) TestInvoiceService(ctx context.Context) (string, error) {
	src := dag.CurrentModule().Source().Directory("..")
	out, err := dag.Container().
		From("golang:1.26-alpine").
		WithExec([]string{"apk", "add", "--no-cache", "nodejs", "npm", "bash", "git"}).
		WithExec([]string{"corepack", "enable"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src/invoice-service").
		WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
		WithExec([]string{"mkdir", "-p", "static"}).
		WithExec([]string{"cp", "node_modules/@picocss/pico/css/pico.min.css", "static/"}).
		WithExec([]string{"cp", "node_modules/htmx.org/dist/htmx.min.js", "static/"}).
		WithExec([]string{"go", "test", "-race", "-count=1", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// BuildAndPublishInvoiceImage builds and publishes invoice-service image.
func (m *Cicd) BuildAndPublishInvoiceImage(ctx context.Context, versionTag string, pushLatest bool) ([]string, error) {
	return m.publishImage(ctx, versionTag, pushLatest, "invoice-service", invoiceImageRepo)
}

// BuildAndPublishOcrImage builds and publishes ocr-service image.
func (m *Cicd) BuildAndPublishOcrImage(ctx context.Context, versionTag string, pushLatest bool) ([]string, error) {
	return m.publishImage(ctx, versionTag, pushLatest, "ocr-service", ocrImageRepo)
}

// Release runs tests and publishes release images for a git tag.
func (m *Cicd) Release(ctx context.Context, tag string) ([]string, error) {
	if !releaseTagPattern.MatchString(tag) {
		return nil, fmt.Errorf("invalid tag %q: expected format v*", tag)
	}
	if _, err := m.TestInvoiceService(ctx); err != nil {
		return nil, fmt.Errorf("invoice-service tests failed: %w", err)
	}

	invoiceRefs, err := m.BuildAndPublishInvoiceImage(ctx, tag, true)
	if err != nil {
		return nil, err
	}
	ocrRefs, err := m.BuildAndPublishOcrImage(ctx, tag, true)
	if err != nil {
		return nil, err
	}

	return append(invoiceRefs, ocrRefs...), nil
}

func (m *Cicd) publishImage(ctx context.Context, versionTag string, pushLatest bool, serviceDir string, repo string) ([]string, error) {
	if !releaseTagPattern.MatchString(versionTag) {
		return nil, fmt.Errorf("invalid release tag %q: expected format v*", versionTag)
	}
	src := dag.CurrentModule().Source().Directory("..").Directory(serviceDir)
	container := src.
		DockerBuild(dagger.DirectoryDockerBuildOpts{Platform: "linux/amd64"})

	publishedRefs := make([]string, 0, 2)
	versionRef, err := container.Publish(ctx, fmt.Sprintf("%s:%s", repo, versionTag))
	if err != nil {
		return nil, fmt.Errorf("failed to publish %s:%s: %w", repo, versionTag, err)
	}
	publishedRefs = append(publishedRefs, versionRef)

	if pushLatest {
		latestRef, err := container.Publish(ctx, fmt.Sprintf("%s:latest", repo))
		if err != nil {
			return nil, fmt.Errorf("failed to publish %s:latest: %w", repo, err)
		}
		publishedRefs = append(publishedRefs, latestRef)
	}

	return publishedRefs, nil
}
