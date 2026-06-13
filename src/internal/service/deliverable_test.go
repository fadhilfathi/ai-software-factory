package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newDeliverableTestService wires a fresh in-memory-backed
// DeliverableService. The store is shared with the caller so
// tests can inspect raw state (e.g. version counts) for the
// "no overwrite of v1" assertion.
func newDeliverableTestService(t *testing.T) (*DeliverableService, store.Store) {
	t.Helper()
	s := store.NewMemoryStore()
	svc := NewDeliverableService(s, zap.NewNop())
	return svc, s
}

// seedDeliverableTaskAndAgent creates a task and an agent in the
// store. projectID is the project the deliverable will live in; the
// task inherits it and the agent is created with the same projectID
// (so cross-tenant checks in the service treat them as same-project).
// Returns (taskID, agentID, projectID). Tests pass projectID through
// to the service as `callerProjectID` for the happy path; cross-tenant
// tests use a different projectID at the call site.
//
// This signature mirrors the seedTaskAndAgent pattern from
// assignment_test.go (TASK-420): the projectID flows through so
// every service call has a `callerProjectID` to compare against.
func seedDeliverableTaskAndAgent(t *testing.T, s store.Store, projectID uuid.UUID) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	task := &model.Task{
		ID:        uuid.New(),
		ProjectID: projectID,
		Title:     "deliv-test-" + uuid.NewString()[:8],
		Status:    model.TaskOpen,
		Priority:  model.PriorityNormal,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Tasks().Create(task))

	agentSvc := NewAgentService(s)
	created, apiErr := agentSvc.CreateAgent(ctx, CreateAgentRequest{
		ProjectID:    task.ProjectID,
		Name:         "agent-" + uuid.NewString()[:8],
		Role:         "developer",
		Capabilities: []string{"coding"},
	})
	require.Nil(t, apiErr)
	return task.ID, created.ID, projectID
}

// expectNoError fails the test if err is non-nil. We use this
// for service-level calls (which return *service.Error) and
// unwrap the assertion for cleaner test code.
func expectNoError(t *testing.T, err *Error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
}

// ----------------------------------------------------------------------------
// CreateDeliverable
// ----------------------------------------------------------------------------

func TestDeliverableService_Create_Success(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   taskID,
		AgentID:  agentID,
		Title:    "My first deliverable",
		Content:  "# Hello\n\nThis is the first version.",
		CreatedBy: nil,
	}, callerProjectID)
	expectNoError(t, svcErr)
	require.NotNil(t, d)
	assert.NotEqual(t, uuid.Nil, d.ID)
	assert.Equal(t, 1, d.Version)
	assert.Equal(t, "My first deliverable", d.Title)
	assert.Equal(t, taskID, d.TaskID)
	assert.Equal(t, agentID, d.AgentID)
	assert.False(t, d.CreatedAt.IsZero())
	assert.False(t, d.UpdatedAt.IsZero())

	// Both rows must exist after Create: the main row + the
	// v1 version row. This is the append-only invariant
	// starting from v1.
	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	assert.Len(t, versions, 1)
	assert.Equal(t, 1, versions[0].Version)
}

// TestDeliverableService_Create_OversizedContent_413 is the
// F-023 service-layer cap. Content > MaxDeliverableContentBytes
// (1 MiB) must be rejected with a 413 PAYLOAD_TOO_LARGE error
// BEFORE any DB read/write is attempted. The 2 MiB payload
// here is comfortably above the cap.
func TestDeliverableService_Create_OversizedContent_413(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	oversize := strings.Repeat("A", int(model.MaxDeliverableContentBytes)+1)
	require.Greater(t, len(oversize), int(model.MaxDeliverableContentBytes))

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   taskID,
		AgentID:  agentID,
		Title:    "oversize-create",
		Content:  oversize,
		CreatedBy: nil,
	}, callerProjectID)
	require.NotNil(t, svcErr, "oversize content must be rejected")
	assert.Nil(t, d)
	assert.Equal(t, http.StatusRequestEntityTooLarge, svcErr.Status)
	assert.Equal(t, "PAYLOAD_TOO_LARGE", svcErr.Code)
}

// TestDeliverableService_Create_AtTheCap_Succeeds documents
// the boundary: a content string exactly at the cap (1 MiB) is
// accepted; one byte over is rejected. This pins the semantics
// of the > comparison.
func TestDeliverableService_Create_AtTheCap_Succeeds(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	atCap := strings.Repeat("B", int(model.MaxDeliverableContentBytes))
	require.Equal(t, int(model.MaxDeliverableContentBytes), len(atCap))

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   taskID,
		AgentID:  agentID,
		Title:    "at-the-cap",
		Content:  atCap,
		CreatedBy: nil,
	}, callerProjectID)
	expectNoError(t, svcErr)
	require.NotNil(t, d)
	assert.Equal(t, 1, d.Version)
}

// TestDeliverableService_Update_OversizedContent_413 covers
// the same cap on the update path. Both Create and Update must
// enforce the limit, otherwise a slow-attacker could PUT a
// giant content repeatedly.
func TestDeliverableService_Update_OversizedContent_413(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	created, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   taskID,
		AgentID:  agentID,
		Title:    "v1",
		Content:  "v1",
		CreatedBy: nil,
	}, callerProjectID)
	expectNoError(t, svcErr)

	oversize := strings.Repeat("A", int(model.MaxDeliverableContentBytes)+1)
	updated, svcErr := svc.UpdateDeliverable(context.Background(), created.ID, UpdateDeliverableRequest{
		Title:   "v2-oversize",
		Content: oversize,
	}, callerProjectID)
	require.NotNil(t, svcErr, "oversize update must be rejected")
	assert.Nil(t, updated)
	assert.Equal(t, http.StatusRequestEntityTooLarge, svcErr.Status)
	assert.Equal(t, "PAYLOAD_TOO_LARGE", svcErr.Code)

	// v1 must be unchanged — the update was rejected before
	// any state change.
	current, svcErr := svc.GetDeliverable(context.Background(), created.ID, callerProjectID)
	expectNoError(t, svcErr)
	assert.Equal(t, 1, current.Version, "v1 must remain after a rejected oversize update")
	assert.Equal(t, "v1", current.Title)
}

func TestDeliverableService_Create_TaskNotFound_404(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	_, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   uuid.New(), // does not exist
		AgentID:  agentID,
		Title:    "x",
		Content:  "y",
	}, callerProjectID)
	assert.Nil(t, d)
	require.NotNil(t, svcErr)
	assert.Equal(t, 404, svcErr.Status)
	assert.Equal(t, "NOT_FOUND", svcErr.Code)
}

func TestDeliverableService_Create_AgentNotFound_404(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, _, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:   taskID,
		AgentID:  uuid.New(), // does not exist
		Title:    "x",
		Content:  "y",
	}, callerProjectID)
	assert.Nil(t, d)
	require.NotNil(t, svcErr)
	assert.Equal(t, 404, svcErr.Status)
}

// ----------------------------------------------------------------------------
// GetDeliverable
// ----------------------------------------------------------------------------

func TestDeliverableService_Get_404(t *testing.T) {
	svc, _ := newDeliverableTestService(t)
	callerProjectID := uuid.New()

	d, svcErr := svc.GetDeliverable(context.Background(), uuid.New(), callerProjectID)
	assert.Nil(t, d)
	require.NotNil(t, svcErr)
	assert.Equal(t, 404, svcErr.Status)
}

// ----------------------------------------------------------------------------
// ListDeliverables
// ----------------------------------------------------------------------------

func TestDeliverableService_List_Pagination(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	// 7 deliverables, all on the same task (so the filter has
	// a single TaskID and we know the expected count).
	for i := 0; i < 7; i++ {
		_, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
			TaskID: taskID, AgentID: agentID,
			Title:   "title-" + uuid.NewString()[:6],
			Content: "body",
		}, callerProjectID)
		expectNoError(t, svcErr)
	}

	// First page: limit=3, no cursor.
	page1, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{
		TaskID: taskID, Limit: 3,
	}, callerProjectID)
	expectNoError(t, svcErr)
	assert.Len(t, page1.Items, 3)
	assert.NotEqual(t, uuid.Nil, page1.NextCursor)

	// Second page.
	page2, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{
		TaskID: taskID, Limit: 3, Cursor: page1.NextCursor,
	}, callerProjectID)
	expectNoError(t, svcErr)
	assert.Len(t, page2.Items, 3)
	assert.NotEqual(t, uuid.Nil, page2.NextCursor)

	// Third page (final): 1 item, no next cursor.
	page3, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{
		TaskID: taskID, Limit: 3, Cursor: page2.NextCursor,
	}, callerProjectID)
	expectNoError(t, svcErr)
	assert.Len(t, page3.Items, 1)
	assert.Equal(t, uuid.Nil, page3.NextCursor)

	// No duplicate IDs across pages.
	seen := map[uuid.UUID]bool{}
	for _, d := range append(append(page1.Items, page2.Items...), page3.Items...) {
		assert.False(t, seen[d.ID])
		seen[d.ID] = true
	}
	assert.Len(t, seen, 7)
}

// ----------------------------------------------------------------------------
// UpdateDeliverable (append-only)
// ----------------------------------------------------------------------------

func TestDeliverableService_Update_CreatesV2(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1 title", Content: "v1 body",
	}, callerProjectID)
	expectNoError(t, svcErr)

	// PUT to v2.
	updated, svcErr := svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
		Title: "v2 title", Content: "v2 body", UpdatedBy: nil,
	}, callerProjectID)
	expectNoError(t, svcErr)
	require.NotNil(t, updated)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, "v2 title", updated.Title)
	assert.Equal(t, "v2 body", updated.Content)

	// Main row at v2.
	raw, svcErr := svc.GetDeliverable(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	assert.Equal(t, 2, raw.Version)

	// Versions table has 2 rows: v1 (original) and v2 (new).
	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	assert.Len(t, versions, 2)

	// v1 still has the original title/content (no overwrite).
	var v1, v2 *model.DeliverableVersion
	for _, v := range versions {
		if v.Version == 1 {
			v1 = v
		} else if v.Version == 2 {
			v2 = v
		}
	}
	require.NotNil(t, v1)
	require.NotNil(t, v2)
	assert.Equal(t, "v1 title", v1.Title)
	assert.Equal(t, "v1 body", v1.Content)
	assert.Equal(t, "v2 title", v2.Title)
	assert.Equal(t, "v2 body", v2.Content)
}

func TestDeliverableService_Update_404(t *testing.T) {
	svc, _ := newDeliverableTestService(t)
	callerProjectID := uuid.New()

	d, svcErr := svc.UpdateDeliverable(context.Background(), uuid.New(), UpdateDeliverableRequest{
		Title: "x", Content: "y",
	}, callerProjectID)
	assert.Nil(t, d)
	require.NotNil(t, svcErr)
	assert.Equal(t, 404, svcErr.Status)
}

// ----------------------------------------------------------------------------
// ListDeliverableVersions
// ----------------------------------------------------------------------------

func TestDeliverableService_ListVersions_404ForMissingDeliverable(t *testing.T) {
	svc, _ := newDeliverableTestService(t)
	callerProjectID := uuid.New()

	versions, svcErr := svc.ListDeliverableVersions(context.Background(), uuid.New(), callerProjectID)
	assert.Nil(t, versions)
	require.NotNil(t, svcErr)
	assert.Equal(t, 404, svcErr.Status)
}

func TestDeliverableService_ListVersions_DESCOrdering(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1", Content: "v1",
	}, callerProjectID)
	expectNoError(t, svcErr)

	// PUT twice → v1, v2, v3.
	for i := 2; i <= 3; i++ {
		_, svcErr := svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
			Title:   "v" + string(rune('0'+i)),
			Content: "v" + string(rune('0'+i)),
		}, callerProjectID)
		expectNoError(t, svcErr)
	}

	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	require.Len(t, versions, 3)
	// DESC: v3, v2, v1.
	assert.Equal(t, 3, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, 1, versions[2].Version)
}

// ----------------------------------------------------------------------------
// Invariants
// ----------------------------------------------------------------------------

// TestDeliverableService_NoOverwriteOfV1AfterV2 verifies the
// append-only invariant: writing v2 must NOT change the v1
// row in deliverable_versions. This is the test that catches
// the regression where someone "optimises" by re-using the
// same row.
func TestDeliverableService_NoOverwriteOfV1AfterV2(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1 title", Content: "v1 body",
	}, callerProjectID)
	expectNoError(t, svcErr)

	// Capture v1's created_at and id for later comparison.
	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	require.Len(t, versions, 1)
	v1ID := versions[0].ID
	v1CreatedAt := versions[0].CreatedAt

	// Sleep 10ms so the v2 created_at is strictly later.
	time.Sleep(10 * time.Millisecond)

	// Write v2.
	_, svcErr = svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
		Title: "v2 title", Content: "v2 body",
	}, callerProjectID)
	expectNoError(t, svcErr)

	// v1 must be unchanged.
	versions, svcErr = svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	require.Len(t, versions, 2)
	var v1After *model.DeliverableVersion
	for _, v := range versions {
		if v.Version == 1 {
			v1After = v
			break
		}
	}
	require.NotNil(t, v1After)
	assert.Equal(t, v1ID, v1After.ID, "v1 row id must be unchanged")
	assert.Equal(t, v1CreatedAt, v1After.CreatedAt, "v1 row created_at must be unchanged")
	assert.Equal(t, "v1 title", v1After.Title)
	assert.Equal(t, "v1 body", v1After.Content)
}

// TestDeliverableService_ConcurrentUpdatesDontRace fires 20
// parallel PUTs at the same deliverable. The expected outcome:
// all but one return 409 (duplicate version) because each
// PUT tries to write the next version (current+1) and
// the version is server-assigned from the current row, so
// they collide on the same version number. The serialised
// "winner" is non-deterministic but exactly one should
// succeed; all others should hit the UNIQUE constraint and
// fail with 409.
//
// This test exercises the WithTx coordination between
// Deliverables().Update and DeliverableVersions().Insert
// and the in-memory store's ErrAlreadyExists on duplicate
// (deliverable_id, version).
func TestDeliverableService_ConcurrentUpdatesDontRace(t *testing.T) {
	svc, s := newDeliverableTestService(t)
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, s, uuid.New())

	d, svcErr := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID: taskID, AgentID: agentID, Title: "v1", Content: "v1",
	}, callerProjectID)
	expectNoError(t, svcErr)

	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	results := make(chan *Error, N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_, svcErr := svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
				Title:   "concurrent",
				Content: "concurrent",
			}, callerProjectID)
			results <- svcErr
		}(i)
	}
	wg.Wait()
	close(results)

	var ok, conflict, other int
	for svcErr := range results {
		if svcErr == nil {
			ok++
		} else if svcErr.Code == "CONFLICT" {
			conflict++
		} else {
			other++
			t.Logf("unexpected error: %+v", svcErr)
		}
	}
	// Exactly one of the N PUTs should win; the rest should
	// hit the (deliverable_id, version) UNIQUE constraint
	// and return 409 CONFLICT. We accept "ok=1 OR ok=0" — the
	// in-memory store doesn't serialise reads inside WithTx,
	// so all N might race on the read of current.Version and
	// all try to write version=2; the first to insert wins
	// and the rest 409. The "other" counter is the
	// canary for a real bug (e.g. a panic inside WithTx).
	assert.Equal(t, 0, other, "no unexpected error codes")
	assert.GreaterOrEqual(t, ok+conflict, N, "all N PUTs should return either ok or CONFLICT")
	assert.GreaterOrEqual(t, ok, 1, "at least one PUT should succeed")
	assert.Equal(t, N, ok+conflict, "no other codes")

	// Sanity: final state is at version=2 (or higher if the
	// winner's write was actually applied, but with one
	// winning insert we expect v2 in the main row).
	final, svcErr := svc.GetDeliverable(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	assert.Equal(t, 2, final.Version, "main row should be at v2 after exactly one winning PUT")

	// The v1 row is still there.
	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, callerProjectID)
	expectNoError(t, svcErr)
	var v1 *model.DeliverableVersion
	for _, v := range versions {
		if v.Version == 1 {
			v1 = v
		}
	}
	require.NotNil(t, v1, "v1 must be preserved")
	assert.Equal(t, "v1", v1.Title)
}


// =============================================================================
// TASK-421 (F-015) cross-tenant deliverable tests
// =============================================================================
//
// Every method must reject:
//   - callerProjectID == uuid.Nil               → 400 MISSING_PROJECT_HEADER
//   - callerProjectID != resource.ProjectID    → 404 CROSS_TENANT_BLOCKED
//
// The defensive triple-check in CreateDeliverable also rejects
// (task.ProjectID != agent.ProjectID) — separate from cross-tenant
// but worth covering.

func TestDeliverableService_Create_CrossTenant_TaskInOtherProject(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	otherProjectID := uuid.New()
	taskID, agentID, _ := seedDeliverableTaskAndAgent(t, store, otherProjectID)

	req := CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "cross-tenant-create",
	}

	callerProjectID := uuid.New() // different from the task/agent project
	d, svcErr := svc.CreateDeliverable(context.Background(), req, callerProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, d)
}

func TestDeliverableService_Create_MissingProjectHeader(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	taskID, agentID, _ := seedDeliverableTaskAndAgent(t, store, uuid.New())

	req := CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "missing-header-create",
	}

	d, svcErr := svc.CreateDeliverable(context.Background(), req, uuid.Nil)
	assertMissingProjectHeader(t, svcErr)
	assert.Nil(t, d)
}

func TestDeliverableService_Get_CrossTenant(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "get-cross-tenant",
	}, callerProjectID)
	require.NotNil(t, d)

	otherProjectID := uuid.New()
	got, svcErr := svc.GetDeliverable(context.Background(), d.ID, otherProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, got)
}

func TestDeliverableService_Get_MissingProjectHeader(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "get-missing-header",
	}, callerProjectID)
	require.NotNil(t, d)

	got, svcErr := svc.GetDeliverable(context.Background(), d.ID, uuid.Nil)
	assertMissingProjectHeader(t, svcErr)
	assert.Nil(t, got)
}

func TestDeliverableService_List_CrossTenant_TaskFilterInOtherProject(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	otherProjectID := uuid.New()
	taskID, _, _ := seedDeliverableTaskAndAgent(t, store, otherProjectID)

	callerProjectID := uuid.New() // does not match task's project
	res, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{
		TaskID: taskID,
	}, callerProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, res)
}

func TestDeliverableService_List_CrossTenant_AgentFilterInOtherProject(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	otherProjectID := uuid.New()
	_, agentID, _ := seedDeliverableTaskAndAgent(t, store, otherProjectID)

	callerProjectID := uuid.New()
	res, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{
		AgentID: agentID,
	}, callerProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, res)
}

func TestDeliverableService_List_MissingProjectHeader(t *testing.T) {
	svc, _ := newDeliverableTestService(t)
	res, svcErr := svc.ListDeliverables(context.Background(), model.DeliverableFilter{}, uuid.Nil)
	assertMissingProjectHeader(t, svcErr)
	assert.Nil(t, res)
}

func TestDeliverableService_Update_CrossTenant(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "update-cross-tenant",
	}, callerProjectID)
	require.NotNil(t, d)

	otherProjectID := uuid.New()
	updated, svcErr := svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
		Title:     "should-not-stick",
		Content:   "should-not-stick",
		UpdatedBy: uuid.New(),
	}, otherProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, updated)
}

func TestDeliverableService_Update_MissingProjectHeader(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "update-missing-header",
	}, callerProjectID)
	require.NotNil(t, d)

	updated, svcErr := svc.UpdateDeliverable(context.Background(), d.ID, UpdateDeliverableRequest{
		Title:     "should-not-stick",
		UpdatedBy: uuid.New(),
	}, uuid.Nil)
	assertMissingProjectHeader(t, svcErr)
	assert.Nil(t, updated)
}

func TestDeliverableService_ListVersions_CrossTenant(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "listversions-cross-tenant",
	}, callerProjectID)
	require.NotNil(t, d)

	otherProjectID := uuid.New()
	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, otherProjectID)
	assertCrossTenantBlocked(t, svcErr)
	assert.Nil(t, versions)
}

func TestDeliverableService_ListVersions_MissingProjectHeader(t *testing.T) {
	store, _ := newDeliverableTestStore(t)
	svc, _ := newDeliverableTestService(t)

	projectID := uuid.New()
	taskID, agentID, callerProjectID := seedDeliverableTaskAndAgent(t, store, projectID)
	d, _ := svc.CreateDeliverable(context.Background(), CreateDeliverableRequest{
		TaskID:  taskID,
		AgentID: agentID,
		Title:   "listversions-missing-header",
	}, callerProjectID)
	require.NotNil(t, d)

	versions, svcErr := svc.ListDeliverableVersions(context.Background(), d.ID, uuid.Nil)
	assertMissingProjectHeader(t, svcErr)
	assert.Nil(t, versions)
}

func assertCrossTenantBlocked(t *testing.T, svcErr *Error) {
	t.Helper()
	require.NotNil(t, svcErr, "expected service.Error, got nil")
	assert.Equal(t, "CROSS_TENANT_BLOCKED", svcErr.Code)
	assert.Equal(t, http.StatusNotFound, svcErr.Status)
}

func assertMissingProjectHeader(t *testing.T, svcErr *Error) {
	t.Helper()
	require.NotNil(t, svcErr, "expected service.Error, got nil")
	assert.Equal(t, "MISSING_PROJECT_HEADER", svcErr.Code)
	assert.Equal(t, http.StatusBadRequest, svcErr.Status)
}
