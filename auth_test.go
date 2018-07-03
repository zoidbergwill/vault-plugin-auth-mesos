package mesosAuthPlugin

import (
	"time"

	"github.com/hashicorp/vault/logical"
)

// See helper_for_test.go for common infrastructure and tools.

// Can't log in without a taskID.
func (ts *TestSuite) Test_login_no_taskID() {
	ts.SetupBackend()
	req := ts.mkReq("login", jsonobj{})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't exist.
func (ts *TestSuite) Test_login_missing_taskID() {
	ts.SetupBackend()
	req := ts.mkReq("login", jsonobj{"task-id": "missing-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't have a Marathon-style app prefix.
func (ts *TestSuite) Test_login_taskID_with_no_prefix() {
	ts.SetupBackend()
	temporarySetOfExistingTasks["abc-123"] = true
	req := ts.mkReq("login", jsonobj{"task-id": "abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can't log in with a taskID that doesn't have policies configured for its
// prefix.
func (ts *TestSuite) Test_login_unregistered_taskID() {
	ts.SetupBackend()
	temporarySetOfExistingTasks["unregistered-task.abc-123"] = true
	req := ts.mkReq("login", jsonobj{"task-id": "unregistered-task.abc-123"})
	ts.HandleRequestError(req, "permission denied")
}

// Can log in with a taskID that exists and has policies configured for its
// prefix.
func (ts *TestSuite) Test_login_good_taskID() {
	ts.SetupBackend()
	temporarySetOfExistingTasks["task-that-exists.abc-123"] = true
	ts.SetTaskPolicies("task-that-exists", "insurance")

	req := ts.mkReq("login", jsonobj{"task-id": "task-that-exists.abc-123"})

	resp := ts.HandleRequest(req)
	ts.Nil(resp.Warnings)
	ts.Nil(resp.Secret)
	ts.Equal(resp.Auth, &logical.Auth{
		Policies:     []string{"insurance"},
		Period:       10 * time.Minute,
		LeaseOptions: logical.LeaseOptions{Renewable: true},
		InternalData: jsonobj{"task-id": "task-that-exists.abc-123"},
	})
}

// Can't renew if you're not logged in.
func (ts *TestSuite) Test_renewal_not_logged_in() {
	ts.SetupBackend()

	req := &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            nil,
		Unauthenticated: false,
	}

	ts.HandleRequestError(req, "request has no secret")
}

// Can renew if you are logged in and your task still exists.
func (ts *TestSuite) Test_renewal_logged_in() {
	ts.SetupBackend()
	temporarySetOfExistingTasks["logged-in-task.abc-123"] = true
	ts.SetTaskPolicies("logged-in-task", "foreign")
	auth := ts.Login("logged-in-task.abc-123")

	req := &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            auth,
		Unauthenticated: false,
	}

	resp := ts.HandleRequest(req)
	ts.Equal(auth, resp.Auth)
}

// Can't renew if your task no longer exists.
func (ts *TestSuite) Test_renewal_task_ended() {
	ts.SetupBackend()
	temporarySetOfExistingTasks["short-task.abc-123"] = true
	ts.SetTaskPolicies("short-task", "foreign")
	auth := ts.Login("short-task.abc-123")
	delete(temporarySetOfExistingTasks, "short-task.abc-123")

	req := &logical.Request{
		Operation:       "renew",
		Path:            "login",
		Auth:            auth,
		Unauthenticated: false,
	}

	ts.HandleRequestError(req, "task short-task.abc-123 not found during renewal")
}
