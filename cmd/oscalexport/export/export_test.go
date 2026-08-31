package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	oscal "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	oscal12 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-2-2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuidance(t *testing.T) {
	tempDir := t.TempDir()
	mockYAML := `
metadata:
  id: Test
  title: Test
  description: ""
groups:
  - id: TEST
    title: Test
    description: Test
guidelines:
  - id: TEST-01
    group: TEST
    title: Test Guideline
`
	inputFilePath := filepath.Join(tempDir, "guidance.yaml")
	require.NoError(t, os.WriteFile(inputFilePath, []byte(mockYAML), 0600))

	t.Run("Success/Defaults", func(t *testing.T) {
		catalogFilePath := filepath.Join(tempDir, "guidance.json")
		profileFilePath := filepath.Join(tempDir, "profile.json")

		args := []string{"--catalog-output", catalogFilePath, "--profile-output", profileFilePath}
		err := Guidance(inputFilePath, args)
		require.NoError(t, err)

		if _, err := os.Stat(catalogFilePath); os.IsNotExist(err) {
			t.Fatalf("Catalog output file not created: %s", catalogFilePath)
		}

		var catalogModel oscal.OscalModels
		catalogData, _ := os.ReadFile(catalogFilePath)
		require.NoError(t, json.Unmarshal(catalogData, &catalogModel))
		assert.NotNil(t, catalogModel.Catalog)

		if _, err := os.Stat(profileFilePath); os.IsNotExist(err) {
			t.Fatalf("Profile output file not created: %s", profileFilePath)
		}

		var profileModel oscal.OscalModels
		profileData, _ := os.ReadFile(profileFilePath)
		require.NoError(t, json.Unmarshal(profileData, &profileModel))
		assert.NotNil(t, profileModel.Profile)
	})

	t.Run("Failure/NotExists", func(t *testing.T) {
		err := Guidance("non-existent-file.yaml", []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Failure/InvalidInput", func(t *testing.T) {
		failYAMLPath := filepath.Join(t.TempDir(), "fail-profile.yaml")
		require.NoError(t, os.WriteFile(failYAMLPath, []byte("fail-profile"), 0600))
		err := Guidance(failYAMLPath, []string{})
		require.ErrorContains(t, err, "string was used where mapping is expected")
	})
}

func TestCatalog(t *testing.T) {
	tempDir := t.TempDir()

	mockYAML := `
metadata:
  id: Test
  title: Test
  description: ""
groups:
  - id: TEST
    title: Test
    description: Test
controls:
  - id: TEST-01
    group: TEST
    title: Test Control
    objective: Test objective
    assessment-requirements:
      - id: TEST-01.1
        text: Test requirement
        applicability: []
`
	inputFilePath := filepath.Join(tempDir, "catalog.yaml")
	require.NoError(t, os.WriteFile(inputFilePath, []byte(mockYAML), 0600))

	t.Run("Success/Defaults", func(t *testing.T) {
		catalogFilePath := filepath.Join(tempDir, "catalog.json")
		args := []string{"--output", catalogFilePath}
		err := Catalog(inputFilePath, args)
		require.NoError(t, err)

		if _, err := os.Stat(catalogFilePath); os.IsNotExist(err) {
			t.Fatalf("Catalog output file not created: %s", catalogFilePath)
		}

		var catalogModel oscal.OscalModels
		catalogData, _ := os.ReadFile(catalogFilePath)
		require.NoError(t, json.Unmarshal(catalogData, &catalogModel))
		assert.NotNil(t, catalogModel.Catalog)
	})

	t.Run("Failure/NotExists", func(t *testing.T) {
		err := Catalog("non-existent-file.yaml", []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Failure/InvalidInput", func(t *testing.T) {
		failYAMLPath := filepath.Join(t.TempDir(), "fail.yaml")
		require.NoError(t, os.WriteFile(failYAMLPath, []byte("fail"), 0600))
		err := Catalog(failYAMLPath, []string{})
		require.ErrorContains(t, err, "string was used where mapping is expected")
	})
}

func TestEvaluation(t *testing.T) {
	tempDir := t.TempDir()

	mockYAML := `
metadata:
  id: test-eval
  type: EvaluationLog
  gemara-version: v1.0.0
  version: "1.0.0"
  date: "2025-01-01T00:00:00Z"
  author:
    name: test-tool
    type: Software
    version: "1.0.0"
result: Failed
target:
  id: sys-1
  name: Test System
  type: Software
evaluations:
  - name: Test Control
    result: Failed
    message: control failed
    control:
      entry-id: CTRL-1
    assessment-logs:
      - requirement:
          entry-id: REQ-1
        description: verify requirement
        result: Failed
        message: requirement not met
        steps-executed: 1
        start: "2025-01-01T00:00:00Z"
`
	inputFilePath := filepath.Join(tempDir, "evaluation.yaml")
	require.NoError(t, os.WriteFile(inputFilePath, []byte(mockYAML), 0600))

	t.Run("Success/Defaults", func(t *testing.T) {
		outputFilePath := filepath.Join(t.TempDir(), "assessment-results.json")
		args := []string{"--output", outputFilePath}
		err := Evaluation(inputFilePath, args)
		require.NoError(t, err)

		var model oscal.OscalModels
		data, err := os.ReadFile(outputFilePath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &model))
		require.NotNil(t, model.AssessmentResults)
		assert.Contains(t, model.AssessmentResults.Metadata.Title, "test-eval")
		require.Len(t, model.AssessmentResults.Results, 1)
	})

	t.Run("Success/WithImportAp", func(t *testing.T) {
		outputFilePath := filepath.Join(t.TempDir(), "assessment-results.json")
		args := []string{"--output", outputFilePath, "--import-ap", "#my-ap"}
		err := Evaluation(inputFilePath, args)
		require.NoError(t, err)

		var model oscal.OscalModels
		data, err := os.ReadFile(outputFilePath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &model))
		assert.Equal(t, "#my-ap", model.AssessmentResults.ImportAp.Href)
	})

	t.Run("Failure/NotExists", func(t *testing.T) {
		err := Evaluation("non-existent-file.yaml", []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Failure/InvalidInput", func(t *testing.T) {
		failYAMLPath := filepath.Join(t.TempDir(), "fail.yaml")
		require.NoError(t, os.WriteFile(failYAMLPath, []byte("fail"), 0600))
		err := Evaluation(failYAMLPath, []string{})
		require.ErrorContains(t, err, "string was used where mapping is expected")
	})

	t.Run("Failure/InvalidCatalogPath", func(t *testing.T) {
		outputFilePath := filepath.Join(t.TempDir(), "assessment-results.json")
		args := []string{"--output", outputFilePath, "--catalog", "non-existent-catalog.yaml"}
		err := Evaluation(inputFilePath, args)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestMapping(t *testing.T) {
	tempDir := t.TempDir()

	mockYAML := `
title: Test Mapping
metadata:
  id: test-mapping
  type: MappingDocument
  gemara-version: "1.2.0"
  version: "1.0.0"
  date: "2026-01-01T00:00:00Z"
  author:
    id: author
    name: Test Author
    type: Human
  mapping-references:
    - id: source-ref
      title: Source Catalog
      version: "1.0.0"
      url: https://example.com/source
    - id: target-ref
      title: Target Catalog
      version: "1.0.0"
      url: https://example.com/target
source-reference:
  entry-type: Control
  reference-id: source-ref
target-reference:
  entry-type: Control
  reference-id: target-ref
mappings:
  - id: map-1
    source: AC-01
    relationship: equivalent
    targets:
      - entry-id: CC-01
        rationale: Same objective
`
	inputFilePath := filepath.Join(tempDir, "mapping.yaml")
	require.NoError(t, os.WriteFile(inputFilePath, []byte(mockYAML), 0600))

	t.Run("Success/Defaults", func(t *testing.T) {
		outputFilePath := filepath.Join(t.TempDir(), "mapping-collection.json")
		args := []string{"--output", outputFilePath}
		err := Mapping(inputFilePath, args)
		require.NoError(t, err)

		var model oscal12.OscalModels
		data, err := os.ReadFile(outputFilePath)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &model))
		require.NotNil(t, model.MappingCollection)
		require.Len(t, model.MappingCollection.Mappings.Maps, 1)
		assert.Equal(t, "equivalent-to", model.MappingCollection.Mappings.Maps[0].Relationship)
	})

	t.Run("Failure/NotExists", func(t *testing.T) {
		err := Mapping("non-existent-file.yaml", []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Failure/InvalidInput", func(t *testing.T) {
		failYAMLPath := filepath.Join(t.TempDir(), "fail.yaml")
		require.NoError(t, os.WriteFile(failYAMLPath, []byte("fail"), 0600))
		err := Mapping(failYAMLPath, []string{})
		require.ErrorContains(t, err, "string was used where mapping is expected")
	})
}
