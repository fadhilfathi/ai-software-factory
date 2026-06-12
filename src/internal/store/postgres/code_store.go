package postgres

import (
	"context"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type postgresCodeStore struct {
	s *postgresStore
}

func (st *postgresCodeStore) CreateCodeGen(req *model.CodeGenRequest) error {
	query := `
		INSERT INTO code_generation_requests (
			id, project_id, task_id, specification, files, status, execution_id, output, estimated_time, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := st.s.pool.Exec(context.Background(), query,
		req.ID, req.ProjectID, req.TaskID, req.Specification, req.Files, req.Status, req.ExecutionID, req.Output, req.EstimatedTime, req.CreatedAt, req.UpdatedAt,
	)
	return err
}

func (st *postgresCodeStore) GetCodeGenByID(id uuid.UUID) (*model.CodeGenRequest, error) {
	query := `
		SELECT id, project_id, task_id, specification, files, status, execution_id, output, estimated_time, created_at, updated_at
		FROM code_generation_requests
		WHERE id = $1`

	var req model.CodeGenRequest
	err := st.s.pool.QueryRow(context.Background(), query, id).Scan(
		&req.ID, &req.ProjectID, &req.TaskID, &req.Specification, &req.Files, &req.Status, &req.ExecutionID, &req.Output, &req.EstimatedTime, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	return &req, err
}

func (st *postgresCodeStore) ListCodeGenByProject(projectID uuid.UUID) ([]*model.CodeGenRequest, error) {
	query := `
		SELECT id, project_id, task_id, specification, files, status, execution_id, output, estimated_time, created_at, updated_at
		FROM code_generation_requests
		WHERE project_id = $1
		ORDER BY created_at DESC`

	rows, err := st.s.pool.Query(context.Background(), query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*model.CodeGenRequest
	for rows.Next() {
		var req model.CodeGenRequest
		err := rows.Scan(
			&req.ID, &req.ProjectID, &req.TaskID, &req.Specification, &req.Files, &req.Status, &req.ExecutionID, &req.Output, &req.EstimatedTime, &req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func (st *postgresCodeStore) UpdateCodeGen(req *model.CodeGenRequest) error {
	query := `
		UPDATE code_generation_requests
		SET status = $2, execution_id = $3, output = $4, estimated_time = $5, updated_at = $6
		WHERE id = $1`

	_, err := st.s.pool.Exec(context.Background(), query,
		req.ID, req.Status, req.ExecutionID, req.Output, req.EstimatedTime, req.UpdatedAt,
	)
	return err
}

func (st *postgresCodeStore) SaveFile(file *model.ProjectFile) error {
	query := `
		INSERT INTO project_files (
			project_id, path, content, language, size, last_modified, modified_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (project_id, path) DO UPDATE
		SET content = EXCLUDED.content, language = EXCLUDED.language, size = EXCLUDED.size, last_modified = EXCLUDED.last_modified, modified_by = EXCLUDED.modified_by`

	_, err := st.s.pool.Exec(context.Background(), query,
		file.ProjectID, file.Path, file.Content, file.Language, file.Size, file.LastModified, file.ModifiedBy,
	)
	return err
}

func (st *postgresCodeStore) GetFile(projectID uuid.UUID, path string) (*model.ProjectFile, error) {
	query := `
		SELECT project_id, path, content, language, size, last_modified, modified_by
		FROM project_files
		WHERE project_id = $1 AND path = $2`

	var file model.ProjectFile
	err := st.s.pool.QueryRow(context.Background(), query, projectID, path).Scan(
		&file.ProjectID, &file.Path, &file.Content, &file.Language, &file.Size, &file.LastModified, &file.ModifiedBy,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	return &file, err
}

func (st *postgresCodeStore) ListFiles(projectID uuid.UUID) ([]*model.ProjectFile, error) {
	query := `
		SELECT project_id, path, content, language, size, last_modified, modified_by
		FROM project_files
		WHERE project_id = $1
		ORDER BY path ASC`

	rows, err := st.s.pool.Query(context.Background(), query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.ProjectFile
	for rows.Next() {
		var file model.ProjectFile
		err := rows.Scan(
			&file.ProjectID, &file.Path, &file.Content, &file.Language, &file.Size, &file.LastModified, &file.ModifiedBy,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	return files, nil
}

func (st *postgresCodeStore) CreateCommit(commit *model.Commit) error {
	ctx := context.Background()
	tx, err := st.s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queryCommit := `
		INSERT INTO commits (sha, project_id, branch, message, author, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(ctx, queryCommit,
		commit.SHA, commit.ProjectID, commit.Branch, commit.Message, commit.Author, commit.CreatedAt,
	)
	if err != nil {
		return err
	}

	queryFile := `
		INSERT INTO commit_files (commit_sha, path, content)
		VALUES ($1, $2, $3)`

	for _, file := range commit.Files {
		_, err = tx.Exec(ctx, queryFile, commit.SHA, file.Path, file.Content)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (st *postgresCodeStore) GetCommit(projectID uuid.UUID, sha string) (*model.Commit, error) {
	query := `
		SELECT sha, project_id, branch, message, author, created_at
		FROM commits
		WHERE project_id = $1 AND sha = $2`

	var commit model.Commit
	err := st.s.pool.QueryRow(context.Background(), query, projectID, sha).Scan(
		&commit.SHA, &commit.ProjectID, &commit.Branch, &commit.Message, &commit.Author, &commit.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Fetch files
	filesQuery := `SELECT path, content FROM commit_files WHERE commit_sha = $1`
	rows, err := st.s.pool.Query(context.Background(), filesQuery, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var file model.CommitFile
		if err := rows.Scan(&file.Path, &file.Content); err != nil {
			return nil, err
		}
		commit.Files = append(commit.Files, file)
	}

	return &commit, nil
}

func (st *postgresCodeStore) ListCommits(projectID uuid.UUID) ([]*model.Commit, error) {
	query := `
		SELECT sha, project_id, branch, message, author, created_at
		FROM commits
		WHERE project_id = $1
		ORDER BY created_at DESC`

	rows, err := st.s.pool.Query(context.Background(), query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []*model.Commit
	for rows.Next() {
		var commit model.Commit
		err := rows.Scan(
			&commit.SHA, &commit.ProjectID, &commit.Branch, &commit.Message, &commit.Author, &commit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		commits = append(commits, &commit)
	}
	return commits, nil
}
