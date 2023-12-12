package release

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/laurentsimon/slsa-policy/pkg/errs"
	"github.com/laurentsimon/slsa-policy/pkg/release/internal/common"
	"github.com/laurentsimon/slsa-policy/pkg/utils/intoto"
)

// TODO: support time creation.
func Test_CreationNew(t *testing.T) {
	t.Parallel()
	subject := intoto.Subject{
		Digests: intoto.DigestSet{
			"sha256":    "some_value",
			"gitCommit": "another_value",
		},
	}
	packageDesc := intoto.ResourceDescriptor{
		URI: "the_uri",
	}
	tests := []struct {
		name          string
		subject       intoto.Subject
		authorVersion string
		buildLevel    *int
		packageDesc   intoto.ResourceDescriptor
		policy        map[string]intoto.Policy
		expected      error
	}{
		{
			name:        "subject and package set",
			subject:     subject,
			packageDesc: packageDesc,
		},
		{
			name:     "result with no package URI",
			subject:  subject,
			expected: errs.ErrorInvalidInput,
		},
		{
			name: "result with no subject digests",
			subject: intoto.Subject{
				URI: "the_uri",
			},
			packageDesc: packageDesc,
			expected:    errs.ErrorInvalidInput,
		},
		{
			name: "result with empty digest value",
			subject: intoto.Subject{
				URI: "the_uri",
				Digests: intoto.DigestSet{
					"sha256":    "some_value",
					"gitCommit": "",
				},
			},
			packageDesc: packageDesc,
			expected:    errs.ErrorInvalidInput,
		},
		{
			name: "result with empty digest key",
			subject: intoto.Subject{
				URI: "the_uri",
				Digests: intoto.DigestSet{
					"sha256": "some_value",
					"":       "another_value",
				},
			},
			packageDesc: packageDesc,
			expected:    errs.ErrorInvalidInput,
		},
		{
			name:          "result with version",
			subject:       subject,
			packageDesc:   packageDesc,
			authorVersion: "my_version",
		},
		{
			name:          "result with author version",
			subject:       subject,
			packageDesc:   packageDesc,
			authorVersion: "my_version",
		},
		{
			name:        "result with level",
			subject:     subject,
			packageDesc: packageDesc,
			buildLevel:  common.AsPointer(2),
		},
		{
			name:        "result with negative level",
			subject:     subject,
			packageDesc: packageDesc,
			buildLevel:  common.AsPointer(-1),
			expected:    errs.ErrorInvalidInput,
		},
		{
			name:        "result with large level",
			subject:     subject,
			packageDesc: packageDesc,
			buildLevel:  common.AsPointer(5),
			expected:    errs.ErrorInvalidInput,
		},
		{
			name:    "result with env",
			subject: subject,
			packageDesc: intoto.ResourceDescriptor{
				URI: "the_uri",
				Annotations: map[string]interface{}{
					environmentAnnotation: "prod",
				},
			},
		},
		{
			name:        "result with policy",
			subject:     subject,
			packageDesc: packageDesc,
			policy: map[string]intoto.Policy{
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
			},
		},
		{
			name:    "result with all set",
			subject: subject,
			packageDesc: intoto.ResourceDescriptor{
				URI: "the_uri",
				Annotations: map[string]interface{}{
					environmentAnnotation: "prod",
				},
			},
			buildLevel:    common.AsPointer(4),
			authorVersion: "my_version",
			policy: map[string]intoto.Policy{
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
			},
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var options []AttestationCreationOption
			if tt.authorVersion != "" {
				options = append(options, SetAuthorVersion(tt.authorVersion))
			}
			if tt.buildLevel != nil {
				options = append(options, SetSlsaBuildLevel(*tt.buildLevel))
			}
			if tt.policy != nil {
				options = append(options, SetPolicy(tt.policy))
			}
			att, err := CreationNew("author_id", tt.subject, tt.packageDesc, options...)
			if diff := cmp.Diff(tt.expected, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			// Statement type verification.
			if diff := cmp.Diff(statementType, att.Header.Type); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// predicate type verification.
			if diff := cmp.Diff(predicateType, att.Header.PredicateType); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// Subjects must match.
			if diff := cmp.Diff([]intoto.Subject{tt.subject}, att.Header.Subjects); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// Author ID must match.
			if diff := cmp.Diff("author_id", att.Predicate.Author.ID); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// Author version must match.
			if diff := cmp.Diff(tt.authorVersion, att.Predicate.Author.Version); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// Policy must match.
			if diff := cmp.Diff(tt.policy, att.Predicate.Policy); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// Package resource must match.
			if diff := cmp.Diff(tt.packageDesc, att.Predicate.Package); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			// SLSA Levels must match.
			if tt.buildLevel != nil {
				if diff := cmp.Diff(*tt.buildLevel, att.Predicate.Properties[buildLevelProperty]); diff != "" {
					t.Fatalf("unexpected err (-want +got): \n%s", diff)
				}
			} else {
				if diff := cmp.Diff(properties(nil), att.Predicate.Properties); diff != "" {
					t.Fatalf("unexpected err (-want +got): \n%s", diff)
				}
			}
		})
	}
}

func Test_EnterSafeMode(t *testing.T) {
	t.Parallel()
	subject := intoto.Subject{
		URI: "the_uri",
		Digests: intoto.DigestSet{
			"sha256":    "some_value",
			"gitCommit": "another_value",
		},
	}
	packageDesc := intoto.ResourceDescriptor{
		URI: "the_uri",
	}
	tests := []struct {
		name        string
		subject     intoto.Subject
		packageDesc intoto.ResourceDescriptor
		options     []AttestationCreationOption
		expected    error
	}{
		{
			name:        "subject only",
			subject:     subject,
			packageDesc: packageDesc,
		},
		{
			name:        "safe mode allowed setters",
			subject:     subject,
			packageDesc: packageDesc,
			options: []AttestationCreationOption{
				EnterSafeMode(),
				SetAuthorVersion("v1.2.3"),
				SetPolicy(map[string]intoto.Policy{
					"org": intoto.Policy{
						URI: "policy1_uri",
					},
				}),
			},
		},
		{
			name:        "safe mode then level",
			subject:     subject,
			packageDesc: packageDesc,
			options: []AttestationCreationOption{
				EnterSafeMode(),
				SetSlsaBuildLevel(4),
			},
			expected: errs.ErrorInternal,
		},
		{
			name:        "level then safe mode",
			subject:     subject,
			packageDesc: packageDesc,
			options: []AttestationCreationOption{
				SetSlsaBuildLevel(4),
				EnterSafeMode(),
			},
		},
		{
			name:        "level then safe mode then allowed setters",
			subject:     subject,
			packageDesc: packageDesc,
			options: []AttestationCreationOption{
				SetSlsaBuildLevel(4),
				EnterSafeMode(),
				SetAuthorVersion("v1.2.3"),
				SetPolicy(map[string]intoto.Policy{
					"org": intoto.Policy{
						URI: "policy1_uri",
					},
				}),
			},
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := CreationNew("author_id", tt.subject, tt.packageDesc, tt.options...)
			if diff := cmp.Diff(tt.expected, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
		})
	}
}
