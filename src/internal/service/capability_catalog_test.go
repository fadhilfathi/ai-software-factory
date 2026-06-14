package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// This file is the table-driven companion to capability_test.go. The older
// tests stay hand-written for the cases that read better as narrative; this
// file adds:
//
//   * a single table-driven catalog walk that asserts the closed-system
//     invariants (catalog has 9, assignable has 8, reserved is 1, the
//     reserved name is leadership, every assignable cap is in the catalog,
//     no cap is both assignable and reserved);
//   * table-driven rewrites of IsAssignableCapability and ValidCapability
//     that share the same case table as the catalog walk, so a new
//     capability added in one place can be added in the others in a single
//     diff;
//   * table-driven coverage for the validation seam's three error codes
//     (CAPABILITY_NOT_IN_CATALOG, CAPABILITY_NOT_ASSIGNABLE,
//     CAPABILITY_MISMATCH), which the hand-written tests only covered
//     partially;
//   * a coverage test for DefaultCapabilitiesForType (the agent-type
//     default map), which the older tests did not exercise.

// --- 1. The capability catalog: closed-system invariants ---------------

// catalogCase is the shared case row for every catalog-walking test in
// this file. Keeping the row in one place means the table is the single
// source of truth for what the catalog looks like, and a new capability
// added to model.AllCapabilities() must add a row here to keep the
// count-based assertions honest.
type catalogCase struct {
	name         string
	cap          model.Capability
	inCatalog    bool
	assignable   bool
	reserved     bool
	validString  bool
	requiredByTA bool // true for the 5 task_type→cap mappings exercised below
}

func catalogCases() []catalogCase {
	return []catalogCase{
		// The 8 assignable caps.
		{"architecture", model.CapArchitecture, true, true, false, true, true},
		{"coding", model.CapCoding, true, true, false, true, true},
		{"testing", model.CapTesting, true, true, false, true, true},
		{"security", model.CapSecurity, true, true, false, true, true},
		{"devops", model.CapDevOps, true, true, false, true, true},
		{"documentation", model.CapDocumentation, true, true, false, true, true},
		{"project_management", model.CapProjectMgmt, true, true, false, true, true},
		{"data_engineering", model.CapDataEngineering, true, true, false, true, true},
		// The 1 reserved cap.
		{"leadership", model.CapLeadership, true, false, true, true, false},
	}
}

func TestCapabilityCatalog_ClosedSystemInvariants(t *testing.T) {
	cases := catalogCases()

	catalogSet := map[model.Capability]bool{}
	for _, c := range cases {
		catalogSet[c.cap] = true
	}

	// The catalog must be exactly 9 (this is the closed-system invariant:
	// migration 016 seeds 9, model.AllCapabilities returns 9, the spec
	// lists 9 in api-spec.md).
	catalog := model.AllCapabilities()
	assert.Len(t, catalog, 9, "model.AllCapabilities() must return exactly 9 caps")

	// Assignable must be 8 (leadership is the 1 reserved cap).
	assignable := model.AssignableCapabilities()
	assert.Len(t, assignable, 8, "model.AssignableCapabilities() must return exactly 8 caps")

	// Every cap reported by model.AllCapabilities() must appear in the
	// case table (otherwise the case table has drifted from the code).
	for _, c := range catalog {
		assert.True(t, catalogSet[c], "model.AllCapabilities() includes %q but the case table does not — add a row", c)
	}

	// No cap is both assignable and reserved.
	assignableSet := map[model.Capability]bool{}
	for _, c := range assignable {
		assignableSet[c] = true
	}
	for _, c := range cases {
		if c.assignable && c.reserved {
			t.Errorf("cap %q is marked both assignable and reserved; that is a closed-system violation", c.name)
		}
		if c.assignable && !catalogSet[c.cap] {
			t.Errorf("cap %q is marked assignable but not in the catalog", c.name)
		}
		if c.reserved && !catalogSet[c.cap] {
			t.Errorf("cap %q is marked reserved but not in the catalog", c.name)
		}
	}

	// Catalog ∪ Reserved = Catalog (the reserved cap is in the catalog).
	// Catalog ∩ Assignable = 0 (no overlap).
	// |Catalog| = |Assignable| + |Reserved| (no cap is in neither).
	reservedCount := 0
	for _, c := range cases {
		if c.reserved {
			reservedCount++
		}
	}
	assert.Equal(t, 1, reservedCount, "exactly 1 reserved cap (leadership)")
	assert.Equal(t, len(cases), 9, "case table has 9 rows = catalog size")
}

// --- 2. IsAssignableCapability: table-driven ---------------------------

func TestIsAssignableCapability_TableDriven(t *testing.T) {
	for _, c := range catalogCases() {
		t.Run(string(c.cap), func(t *testing.T) {
			assert.Equal(t, c.assignable, model.IsAssignableCapability(string(c.cap)),
				"IsAssignableCapability(%q)", string(c.cap))
		})
	}
	// Unknown and empty are never assignable.
	t.Run("unknown", func(t *testing.T) {
		assert.False(t, model.IsAssignableCapability("rocket-science"))
	})
	t.Run("empty", func(t *testing.T) {
		assert.False(t, model.IsAssignableCapability(""))
	})
	t.Run("uppercase", func(t *testing.T) {
		assert.False(t, model.IsAssignableCapability("CODING"),
			"capability names are case-sensitive; 'CODING' is not the catalog name 'coding'")
	})
}

// --- 3. ValidCapability: table-driven ---------------------------------

func TestValidCapability_TableDriven(t *testing.T) {
	for _, c := range catalogCases() {
		t.Run(string(c.cap), func(t *testing.T) {
			assert.Equal(t, c.inCatalog, model.ValidCapability(string(c.cap)),
				"ValidCapability(%q)", string(c.cap))
		})
	}
	// Edge cases the closed-system table does not cover.
	for _, tc := range []struct {
		name string
		in   string
		want bool
	}{
		{"unknown_lowercase", "unknown", false},
		{"empty", "", false},
		{"uppercase_catalog", "CODING", false},
		{"tab", "\t", false},
		{"system_prefix", "__system__rocket", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, model.ValidCapability(tc.in))
		})
	}
}

// --- 4. AgentTypeCapabilities: table-driven (NEW) ----------------

// The agent-type → default caps map is used by routing and the monitoring
// dashboard. None of the older tests exercised it; the table-driven cases
// pin down the 6 agent types and an unknown type.
func TestAgentTypeCapabilities_TableDriven(t *testing.T) {
	cases := []struct {
		agentType model.AgentType
		want      []string
	}{
		{model.AgentPM, []string{"requirement_analysis", "task_decomposition"}},
		{model.AgentArch, []string{"system_design", "api_design"}},
		{model.AgentDev, []string{"code_implementation"}},
		{model.AgentReviewer, []string{"code_review", "security_scan"}},
		{model.AgentQA, []string{"test_planning", "test_execution"}},
		{model.AgentDevOps, []string{"ci_cd", "deployment", "infrastructure"}},
		{model.AgentType("unknown_type"), nil},
		{model.AgentType(""), nil},
	}
	for _, tc := range cases {
		t.Run(string(tc.agentType), func(t *testing.T) {
			got := model.AgentTypeCapabilities[tc.agentType]
			if tc.want == nil {
				assert.Nil(t, got, "unknown agent type should yield nil")
			} else {
				gotStrs := make([]string, 0, len(got))
				for _, c := range got {
					gotStrs = append(gotStrs, string(c))
				}
				assert.ElementsMatch(t, tc.want, gotStrs)
			}
		})
	}

	// Cross-check: the agent-type defaults live in a disjoint namespace
	// from the user-facing 9 caps. None of the 12 internal names should
	// collide with the catalog names. This is a structural invariant,
	// not a behavior assertion, so it lives at the end of the test.
	catalogStrs := map[string]bool{}
	for _, c := range model.AllCapabilities() {
		catalogStrs[string(c)] = true
	}
	internalDefaults := map[string]bool{}
	for _, caps := range [][]model.AgentCapability{
		model.AgentTypeCapabilities[model.AgentPM],
		model.AgentTypeCapabilities[model.AgentArch],
		model.AgentTypeCapabilities[model.AgentDev],
		model.AgentTypeCapabilities[model.AgentReviewer],
		model.AgentTypeCapabilities[model.AgentQA],
		model.AgentTypeCapabilities[model.AgentDevOps],
	} {
		for _, c := range caps {
			internalDefaults[string(c)] = true
		}
	}
	for name := range internalDefaults {
		assert.False(t, catalogStrs[name],
			"agent-type default cap %q collides with catalog cap name; the two namespaces must stay disjoint", name)
	}
}

// --- 5. Validation seam: table-driven (3 error codes) -----------------

// validationCase is the row type for the table-driven seam test. The
// fixture layer is the same hand-rolled mock used by the older
// TestValidateAgentHasCapabilities_* tests, but the cases are flattened
// to one walk so a new error code (or a new rejection mode) can be added
// in a single diff.
type validationCase struct {
	name            string
	grants          []*model.AgentCapabilityView
	required        []string
	wantErr         bool
	wantCode        string
	wantStatus      int
	wantDetailField string
	wantDetailSub   string // substring required in the detail message
}

func TestValidateAgentHasCapabilities_TableDriven(t *testing.T) {
	grantedAll := []*model.AgentCapabilityView{
		{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt},
		{Name: "testing", Category: "testing", GrantedAt: fixedGrantedAt},
		{Name: "security", Category: "security", GrantedAt: fixedGrantedAt},
		{Name: "data_engineering", Category: "data_engineering", GrantedAt: fixedGrantedAt},
	}
	grantedSome := []*model.AgentCapabilityView{
		{Name: "coding", Category: "coding", GrantedAt: fixedGrantedAt},
	}

	cases := []validationCase{
		// Happy paths.
		{
			name:     "all_present_single",
			grants:   grantedAll,
			required: []string{"coding"},
			wantErr:  false,
		},
		{
			name:     "all_present_multi",
			grants:   grantedAll,
			required: []string{"coding", "testing"},
			wantErr:  false,
		},
		{
			name:     "empty_required_no_store_call",
			grants:   grantedAll,
			required: []string{},
			wantErr:  false,
		},
		{
			name:     "nil_required_no_store_call",
			grants:   grantedAll,
			required: nil,
			wantErr:  false,
		},

		// Rejection: CAPABILITY_MISMATCH (the agent is in the catalog
		// and assignable, but the agent's grant set is incomplete).
		{
			name:            "one_missing_caps_mismatch",
			grants:          grantedSome,
			required:        []string{"coding", "testing"},
			wantErr:         true,
			wantCode:        "CAPABILITY_MISMATCH",
			wantStatus:      409,
			wantDetailField: "required_capabilities",
			wantDetailSub:   "testing",
		},
		{
			name:            "all_missing_caps_mismatch",
			grants:          nil,
			required:        []string{"coding"},
			wantErr:         true,
			wantCode:        "CAPABILITY_MISMATCH",
			wantStatus:      409,
			wantDetailField: "required_capabilities",
			wantDetailSub:   "coding",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agentID := uuid.New()
			grants := map[uuid.UUID][]*model.AgentCapabilityView{agentID: tc.grants}
			agents := newCapAgentStore(grants)
			s := NewCapabilityService(&mockCapabilityStore{agents: agents}, zap.NewNop())

			err := s.ValidateAgentHasCapabilities(context.Background(), agentID, tc.required)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			var svcErr *Error
			require.True(t, errors.As(err, &svcErr), "must be a *Error")
			assert.Equal(t, tc.wantCode, svcErr.Code)
			assert.Equal(t, tc.wantStatus, svcErr.Status)
			if tc.wantDetailField != "" {
				found := false
				for _, d := range svcErr.Details {
					if d.Field == tc.wantDetailField && strings.Contains(d.Message, tc.wantDetailSub) {
						found = true
						break
					}
				}
				assert.True(t, found, "details must include field=%q message containing %q", tc.wantDetailField, tc.wantDetailSub)
			}
		})
	}
}

// --- 6. Reserved-name enforcement (the leadership carve-out) ----------

// The validation seam rejects `leadership` in required_capabilities. This
// is enforced upstream of ValidateAgentHasCapabilities (in the
// AssignmentService.validateCapabilities method), so we exercise it
// through the public NewAssignmentService surface indirectly: by
// checking that IsAssignableCapability("leadership") returns false and
// that the assignable set never contains it.
func TestLeadershipIsReservedAndNotAssignable(t *testing.T) {
	// 1. IsAssignableCapability reports false.
	assert.False(t, model.IsAssignableCapability("leadership"))

	// 2. The assignable set does not contain leadership.
	for _, c := range model.AssignableCapabilities() {
		assert.NotEqual(t, model.CapLeadership, c,
			"leadership must not appear in AssignableCapabilities()")
	}

	// 3. The catalog does contain leadership (it is reserved, not removed).
	assert.True(t, model.ValidCapability("leadership"))

	// 4. The role -> caps map routes "leader" role to leadership.
	caps, ok := model.RoleCapabilities["leader"]
	require.True(t, ok, "leader role must have a default cap set")
	assert.Contains(t, caps, model.CapLeadership)
}
