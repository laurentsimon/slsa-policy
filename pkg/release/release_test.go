package release

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/laurentsimon/slsa-policy/pkg/errs"
	"github.com/laurentsimon/slsa-policy/pkg/release/internal/common"
	"github.com/laurentsimon/slsa-policy/pkg/release/internal/organization"
	"github.com/laurentsimon/slsa-policy/pkg/release/internal/project"
	"github.com/laurentsimon/slsa-policy/pkg/utils/intoto"
)

func Test_AttestationNew(t *testing.T) {
	t.Parallel()
	digests := intoto.DigestSet{
		"sha256":    "some_value",
		"gitCommit": "another_value",
	}
	subject := intoto.Subject{
		Digests: digests,
	}
	level := 2
	packageURI := "package_uri"
	environment := common.AsPointer("prod")
	authorVersion := "v1.2.3"
	policy := map[string]intoto.Policy{
		"org": intoto.Policy{
			URI: "policy1_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value1",
				"commitSha": "value2",
			},
		},
		"project": intoto.Policy{
			URI: "policy2_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value3",
				"commitSha": "value4",
			},
		},
	}
	authorID := "author_id"
	tests := []struct {
		name          string
		authorID      string
		result        PolicyEvaluationResult
		options       []AttestationCreationOption
		subject       intoto.Subject
		authorVersion string
		policy        map[string]intoto.Policy
		level         int
		expected      error
	}{
		{
			name:     "all fields set",
			authorID: authorID,
			result: PolicyEvaluationResult{
				level:       level,
				packageURI:  packageURI,
				digests:     digests,
				environment: environment,
			},
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			subject:       subject,
			level:         level,
			authorVersion: authorVersion,
			policy:        policy,
		},
		{
			name:     "no env",
			authorID: authorID,
			result: PolicyEvaluationResult{
				level:      level,
				packageURI: packageURI,
				digests:    digests,
			},
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			subject:       subject,
			level:         level,
			authorVersion: authorVersion,
			policy:        policy,
		},
		{
			name:     "no author version",
			authorID: authorID,
			result: PolicyEvaluationResult{
				level:       level,
				packageURI:  packageURI,
				digests:     digests,
				environment: environment,
			},
			options: []AttestationCreationOption{
				SetPolicy(policy),
			},
			subject: subject,
			level:   level,
			policy:  policy,
		},
		{
			name:     "no policy",
			authorID: authorID,
			result: PolicyEvaluationResult{
				level:       level,
				packageURI:  packageURI,
				digests:     digests,
				environment: environment,
			},
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
			},
			subject:       subject,
			level:         level,
			authorVersion: authorVersion,
		},
		{
			name:     "error result",
			authorID: authorID,
			result: PolicyEvaluationResult{
				err: errs.ErrorMismatch,
			},
			expected: errs.ErrorInternal,
		},
		{
			name:     "invalid result",
			authorID: authorID,
			expected: errs.ErrorInternal,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			att, err := tt.result.AttestationNew(tt.authorID, tt.options...)
			if diff := cmp.Diff(tt.expected, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.authorID, att.attestation.Predicate.Author.ID); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff([]intoto.Subject{tt.subject}, att.attestation.Header.Subjects); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.result.packageURI, att.attestation.Predicate.Package.URI); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			properties := att.attestation.Predicate.Properties
			val, exists := properties[buildLevelProperty]
			if !exists {
				t.Fatalf("%q property does not exist: \n", buildLevelProperty)
			}
			v, ok := val.(int)
			if !ok {
				t.Fatalf("%q is not an int: %T\n", val, val)
			}
			if diff := cmp.Diff(tt.level, v); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			var expectedEnv string
			if tt.result.environment != nil {
				expectedEnv = *tt.result.environment
			}
			env, err := intoto.GetAnnotationValue(att.attestation.Predicate.Package.Annotations, environmentAnnotation)
			if err != nil {
				t.Fatalf("failed to retrieve annotation: %v\n", err)
			}
			if diff := cmp.Diff(expectedEnv, env); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.authorVersion, att.attestation.Predicate.Author.Version); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.policy, att.attestation.Predicate.Policy); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
		})
	}
}

func Test_e2e(t *testing.T) {
	t.Parallel()
	digests := intoto.DigestSet{
		"sha256":    "some_value",
		"gitCommit": "another_value",
	}
	subject := intoto.Subject{
		Digests: digests,
	}

	packageURI := "package_uri"
	packageURI1 := "package_uri1"
	environment := common.AsPointer("prod")
	authorVersion := "v1.2.3"
	policy := map[string]intoto.Policy{
		"org": intoto.Policy{
			URI: "policy1_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value1",
				"commitSha": "value2",
			},
		},
		"project": intoto.Policy{
			URI: "policy2_uri",
			Digests: intoto.DigestSet{
				"sha256":    "value3",
				"commitSha": "value4",
			},
		},
	}
	selfHostedRunner := "https://github.com/actions/runner/self-hosted"
	githubHostedRunner := "https://github.com/actions/runner/github-hosted"
	selfLevel := 2
	githubLevel := 3
	authorID := "author_id"
	sourceURI := "source_uri"
	sourceURI1 := "source_uri1"
	orgPolicy := organization.Policy{
		Format: 1,
		Roots: organization.Roots{
			Build: []organization.Root{
				{
					ID:        githubHostedRunner,
					Name:      "github_actions_level_3",
					SlsaLevel: common.AsPointer(githubLevel),
				},
				{
					ID:        selfHostedRunner,
					Name:      "github_actions_level_2",
					SlsaLevel: common.AsPointer(selfLevel),
				},
			},
		},
	}
	projectsPolicy := []project.Policy{
		{
			Format: 1,
			Package: project.Package{
				URI: packageURI,
				Environment: project.Environment{
					AnyOf: []string{"dev", "prod"},
				},
			},
			BuildRequirements: project.BuildRequirements{
				RequireSlsaBuilder: "github_actions_level_3",
				Repository: project.Repository{
					URI: sourceURI,
				},
			},
		},
		{
			Format: 1,
			Package: project.Package{
				URI: packageURI1,
			},
			BuildRequirements: project.BuildRequirements{
				RequireSlsaBuilder: "github_actions_level_2",
				Repository: project.Repository{
					URI: sourceURI1,
				},
			},
		},
	}
	tests := []struct {
		name             string
		authorID         string
		org              organization.Policy
		projects         []project.Policy
		options          []AttestationCreationOption
		subject          intoto.Subject
		environment      *string
		packageURI       string
		authorVersion    string
		policy           map[string]intoto.Policy
		level            int
		builderID        string
		sourceURI        string
		errorEvaluate    error
		errorAttestation error
	}{
		{
			name:     "all fields set",
			authorID: authorID,
			// Policies to evaluate.
			org:         orgPolicy,
			projects:    projectsPolicy,
			environment: environment,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID: githubHostedRunner,
			sourceURI: sourceURI,
		},
		{
			name:     "env not provided",
			authorID: authorID,
			// Policies to evaluate.
			org:      orgPolicy,
			projects: projectsPolicy,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID:        githubHostedRunner,
			sourceURI:        sourceURI,
			errorEvaluate:    errs.ErrorInvalidInput,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name:     "env not in policy",
			authorID: authorID,
			// Policies to evaluate.
			org: orgPolicy,
			projects: []project.Policy{
				{
					Format: 1,
					Package: project.Package{
						URI: packageURI,
					},
					BuildRequirements: project.BuildRequirements{
						RequireSlsaBuilder: "github_actions_level_3",
						Repository: project.Repository{
							URI: sourceURI,
						},
					},
				},
			},
			environment: environment,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID:        githubHostedRunner,
			sourceURI:        sourceURI,
			errorEvaluate:    errs.ErrorInvalidInput,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name:     "mismatch env",
			authorID: authorID,
			// Policies to evaluate.
			org:         orgPolicy,
			projects:    projectsPolicy,
			environment: common.AsPointer("not_prod"),
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID:        githubHostedRunner,
			sourceURI:        sourceURI,
			errorEvaluate:    errs.ErrorNotFound,
			errorAttestation: errs.ErrorInternal,
		},
		{
			name:     "no env",
			authorID: authorID,
			// Policies to evaluate.
			org: orgPolicy,
			projects: []project.Policy{
				{
					Format: 1,
					Package: project.Package{
						URI: packageURI,
					},
					BuildRequirements: project.BuildRequirements{
						RequireSlsaBuilder: "github_actions_level_3",
						Repository: project.Repository{
							URI: sourceURI,
						},
					},
				},
			},
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID: githubHostedRunner,
			sourceURI: sourceURI,
		},
		{
			name:     "not autor version",
			authorID: authorID,
			// Policies to evaluate.
			org:         orgPolicy,
			projects:    projectsPolicy,
			environment: environment,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject: subject,
			level:   githubLevel,
			policy:  policy,
			// Builder that the verifier will use.
			builderID: githubHostedRunner,
			sourceURI: sourceURI,
		},
		{
			name:     "no policy",
			authorID: authorID,
			// Policies to evaluate.
			org:         orgPolicy,
			projects:    projectsPolicy,
			environment: environment,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			// Builder that the verifier will use.
			builderID: githubHostedRunner,
			sourceURI: sourceURI,
		},
		{
			name:     "evaluation error",
			authorID: authorID,
			// Policies to evaluate.
			org:      orgPolicy,
			projects: projectsPolicy,
			// Options to create the attestation.
			options: []AttestationCreationOption{
				SetAuthorVersion(authorVersion),
				SetPolicy(policy),
			},
			packageURI: packageURI,
			// Fields to validate the created attestation.
			subject:       subject,
			level:         githubLevel,
			authorVersion: authorVersion,
			policy:        policy,
			// Builder that the verifier will use.
			builderID:        githubHostedRunner,
			sourceURI:        sourceURI,
			errorEvaluate:    errs.ErrorInvalidInput,
			errorAttestation: errs.ErrorInternal,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create the reader for the org policy.
			orgContent, err := json.Marshal(tt.org)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			orgReader := io.NopCloser(bytes.NewReader(orgContent))
			// Create the readers for the projects policy.
			// Marshal the project policies into bytes.
			policies := make([][]byte, len(tt.projects), len(tt.projects))
			for i := range tt.projects {
				content, err := json.Marshal(tt.projects[i])
				if err != nil {
					t.Fatalf("failed to marshal: %v", err)
				}
				policies[i] = content
			}
			projectsReader := common.NewBytesIterator(policies)
			pol, err := PolicyNew(orgReader, projectsReader)
			if err != nil {
				t.Fatalf("failed to create policy: %v", err)
			}
			verifier := common.NewAttestationVerifier(tt.subject.Digests, tt.packageURI, tt.builderID, tt.sourceURI)
			opts := BuildVerificationOption{
				Verifier:    verifier,
				Environment: tt.environment,
			}
			result := pol.Evaluate(tt.subject.Digests, tt.packageURI, opts)
			if diff := cmp.Diff(tt.errorEvaluate, result.Error(), cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			att, err := result.AttestationNew(tt.authorID, tt.options...)
			if diff := cmp.Diff(tt.errorAttestation, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.authorID, att.attestation.Predicate.Author.ID); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff([]intoto.Subject{tt.subject}, att.attestation.Header.Subjects); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(result.packageURI, att.attestation.Predicate.Package.URI); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			properties := att.attestation.Predicate.Properties
			val, exists := properties[buildLevelProperty]
			if !exists {
				t.Fatalf("%q property does not exist: \n", buildLevelProperty)
			}
			v, ok := val.(int)
			if !ok {
				t.Fatalf("%q is not an int: %T\n", val, val)
			}
			if diff := cmp.Diff(tt.level, v); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			var expectedEnv string
			if tt.environment != nil {
				expectedEnv = *tt.environment
			}
			env, err := intoto.GetAnnotationValue(att.attestation.Predicate.Package.Annotations, environmentAnnotation)
			if err != nil {
				t.Fatalf("failed to retrieve annotation: %v\n", err)
			}
			if diff := cmp.Diff(expectedEnv, env); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.authorVersion, att.attestation.Predicate.Author.Version); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if diff := cmp.Diff(tt.policy, att.attestation.Predicate.Policy); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
		})
	}
}
