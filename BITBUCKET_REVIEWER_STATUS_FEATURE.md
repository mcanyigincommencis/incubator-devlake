# Bitbucket Pull Request Reviewer Status Feature

## Overview

This document describes the implementation of Pull Request reviewer status collection for the Bitbucket Cloud plugin. This feature enables collection of reviewer information including approval status and change requests, which can be used to calculate metrics like "Review Quality" and "Change Request Rate".

## Changes Made

### 1. New Model: `BitbucketPrReviewer`

**File:** `backend/plugins/bitbucket/models/pr_reviewer.go`

A new model has been created to store PR reviewer information:

```go
type BitbucketPrReviewer struct {
    ConnectionId    uint64
    RepoId          string
    PullRequestId   int
    ReviewerAccount string  // account_id or uuid of the reviewer
    ReviewerName    string  // Display name of the reviewer
    Role            string  // e.g., "REVIEWER", "PARTICIPANT"
    Approved        bool    // Whether the reviewer approved the PR
    State           string  // e.g., "approved", "changes_requested", null
    ParticipatedOn  *time.Time
}
```

**Table:** `_tool_bitbucket_pull_request_reviewers`

### 2. Updated PR Collector

**File:** `backend/plugins/bitbucket/tasks/pr_collector.go`

The collector now requests `participants` data from the Bitbucket API:

```
values.participants.user
values.participants.role
values.participants.approved
values.participants.state
values.participants.participated_on
```

### 3. Updated PR Extractor

**File:** `backend/plugins/bitbucket/tasks/pr_extractor.go`

The extractor now processes the `participants` array from the API response:

- Extracts reviewer account information
- Creates `BitbucketPrReviewer` records for each participant
- Links reviewers to their respective pull requests

### 4. Database Migration

**File:** `backend/plugins/bitbucket/models/migrationscripts/20251218_add_pr_reviewers_table.go`

A new migration script creates the `_tool_bitbucket_pull_request_reviewers` table with the following schema:

**Primary Key:** (`connection_id`, `repo_id`, `pull_request_id`, `reviewer_account`)

**Columns:**
- `connection_id` (uint64)
- `repo_id` (varchar(255))
- `pull_request_id` (int)
- `reviewer_account` (varchar(255))
- `reviewer_name` (varchar(255))
- `role` (varchar(100))
- `approved` (bool)
- `state` (varchar(100))
- `participated_on` (timestamp)
- Standard audit fields (`created_at`, `updated_at`)

### 5. Updated E2E Tests

**File:** `backend/plugins/bitbucket/e2e/pr_test.go`

The E2E test now verifies the reviewer table is populated correctly.

## Usage

### Querying Reviewer Data

After running the Bitbucket plugin with this feature, you can query reviewer information:

```sql
-- Get all reviewers for a specific PR
SELECT 
    pull_request_id,
    reviewer_name,
    role,
    approved,
    state,
    participated_on
FROM _tool_bitbucket_pull_request_reviewers
WHERE connection_id = 1 
  AND repo_id = 'owner/repo'
  AND pull_request_id = 123;
```

### Calculating Metrics

#### Review Quality Metric

Count PRs where all reviewers approved:

```sql
SELECT 
    pr.bitbucket_id,
    pr.title,
    COUNT(r.reviewer_account) as total_reviewers,
    SUM(CASE WHEN r.approved = true THEN 1 ELSE 0 END) as approved_count
FROM _tool_bitbucket_pull_requests pr
LEFT JOIN _tool_bitbucket_pull_request_reviewers r 
    ON pr.connection_id = r.connection_id 
    AND pr.repo_id = r.repo_id 
    AND pr.bitbucket_id = r.pull_request_id
WHERE pr.state = 'MERGED'
GROUP BY pr.bitbucket_id, pr.title;
```

#### Change Request Rate

Calculate the percentage of PRs that received change requests:

```sql
SELECT 
    COUNT(DISTINCT pr.bitbucket_id) as total_prs,
    COUNT(DISTINCT CASE 
        WHEN r.state = 'changes_requested' THEN pr.bitbucket_id 
    END) as prs_with_changes_requested,
    (COUNT(DISTINCT CASE 
        WHEN r.state = 'changes_requested' THEN pr.bitbucket_id 
    END) * 100.0 / COUNT(DISTINCT pr.bitbucket_id)) as change_request_rate
FROM _tool_bitbucket_pull_requests pr
LEFT JOIN _tool_bitbucket_pull_request_reviewers r 
    ON pr.connection_id = r.connection_id 
    AND pr.repo_id = r.repo_id 
    AND pr.bitbucket_id = r.pull_request_id
WHERE pr.state = 'MERGED';
```

#### Review Participation Rate

Calculate average number of reviewers per PR:

```sql
SELECT 
    AVG(reviewer_count) as avg_reviewers_per_pr
FROM (
    SELECT 
        pr.bitbucket_id,
        COUNT(r.reviewer_account) as reviewer_count
    FROM _tool_bitbucket_pull_requests pr
    LEFT JOIN _tool_bitbucket_pull_request_reviewers r 
        ON pr.connection_id = r.connection_id 
        AND pr.repo_id = r.repo_id 
        AND pr.bitbucket_id = r.pull_request_id
    GROUP BY pr.bitbucket_id
) pr_reviews;
```

## Bitbucket API Reference

This feature uses the Bitbucket Cloud API v2.0 `participants` field:

- **API Endpoint:** `GET /2.0/repositories/{workspace}/{repo_slug}/pullrequests`
- **Documentation:** https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/

### Participant Object Structure

```json
{
  "participants": [
    {
      "user": {
        "display_name": "John Doe",
        "account_id": "557058:12345678-1234-1234-1234-123456789012",
        "uuid": "{12345678-1234-1234-1234-123456789012}"
      },
      "role": "REVIEWER",
      "approved": true,
      "state": "approved",
      "participated_on": "2025-12-18T10:30:00.000000+00:00"
    }
  ]
}
```

### Possible Values

**Role:**
- `REVIEWER` - User was explicitly added as a reviewer
- `PARTICIPANT` - User participated in the PR (commented, approved, etc.)

**State:**
- `approved` - Reviewer approved the PR
- `changes_requested` - Reviewer requested changes
- `null` - No explicit review state (just participated)

**Approved:**
- `true` - User has approved the PR
- `false` - User has not approved

## Migration

When upgrading to this version:

1. The migration will automatically create the new table
2. Existing PRs will not have reviewer data
3. Re-run the Bitbucket data collection to populate reviewer information for existing PRs

## Backward Compatibility

This feature is fully backward compatible:
- Existing PRs continue to work as before
- The new table is optional and doesn't affect existing queries
- If the API doesn't return participants data, the plugin continues to work without errors

## Testing

Run the E2E test to verify the feature:

```bash
cd backend
go test -v ./plugins/bitbucket/e2e -run TestPrDataFlow
```

## Troubleshooting

### No Reviewer Data Appearing

1. **Check API Response:** Ensure the Bitbucket API is returning `participants` data
2. **Verify Query Parameters:** The collector must include participants fields in the query
3. **Check Migration:** Ensure the migration ran successfully and created the table

### Missing Participants in Raw Data

The Bitbucket API only returns participants when explicitly requested via the `fields` query parameter. Ensure the collector includes:

```
values.participants.user,values.participants.role,values.participants.approved,values.participants.state,values.participants.participated_on
```

## Future Enhancements

Potential improvements for this feature:

1. **Review Comments:** Link reviewer status to specific comments
2. **Review History:** Track changes in reviewer status over time
3. **Domain Layer Mapping:** Create domain layer entities for reviewers
4. **Grafana Dashboards:** Pre-built dashboards for review metrics
5. **Review Time Metrics:** Calculate time between PR creation and first review

## References

- [Bitbucket Cloud REST API Documentation](https://developer.atlassian.com/cloud/bitbucket/rest/)
- [DevLake Plugin Development Guide](https://devlake.apache.org/docs/DeveloperManuals/PluginImplementation)


