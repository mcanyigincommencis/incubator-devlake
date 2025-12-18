/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

var _ plugin.MigrationScript = (*addPrReviewersTable)(nil)

type bitbucketPrReviewer20251218 struct {
	ConnectionId    uint64 `gorm:"primaryKey"`
	RepoId          string `gorm:"primaryKey;type:varchar(255)"`
	PullRequestId   int    `gorm:"primaryKey;autoIncrement:false"`
	ReviewerAccount string `gorm:"primaryKey;type:varchar(255)"`
	ReviewerName    string `gorm:"type:varchar(255)"`
	Role            string `gorm:"type:varchar(100)"`
	Approved        bool
	State           string `gorm:"type:varchar(100)"`
	ParticipatedOn  *time.Time
	common.NoPKModel
}

func (bitbucketPrReviewer20251218) TableName() string {
	return "_tool_bitbucket_pull_request_reviewers"
}

type addPrReviewersTable struct{}

func (script *addPrReviewersTable) Up(basicRes context.BasicRes) errors.Error {
	// Create the new table for PR reviewers
	return migrationhelper.AutoMigrateTables(basicRes, &bitbucketPrReviewer20251218{})
}

func (*addPrReviewersTable) Version() uint64 {
	return 20251218000001
}

func (script *addPrReviewersTable) Name() string {
	return "add PR reviewers table to track review status (approved/changes_requested)"
}


