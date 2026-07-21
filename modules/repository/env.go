// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"os"
	"strconv"
	"strings"

	repo_model "forgente.com/models/repo"
	user_model "forgente.com/models/user"
	"forgente.com/modules/setting"
	"forgente.com/modules/util"
)

// env keys for git hooks need
// The GITEA_* names are kept forever: they are read by users' own custom git hooks, so removing
// them would silently break third-party scripts. FORGENTE_* siblings are dual-emitted alongside
// them; our own code (see EnvGet below) reads the FORGENTE_* name first, falling back to GITEA_*.
const (
	EnvRepoName      = "GITEA_REPO_NAME"
	EnvRepoUsername  = "GITEA_REPO_USER_NAME"
	EnvRepoID        = "GITEA_REPO_ID"
	EnvRepoIsWiki    = "GITEA_REPO_IS_WIKI"
	EnvPusherName    = "GITEA_PUSHER_NAME"
	EnvPusherEmail   = "GITEA_PUSHER_EMAIL"
	EnvPusherID      = "GITEA_PUSHER_ID"
	EnvKeyID         = "GITEA_KEY_ID" // public key ID
	EnvDeployKeyID   = "GITEA_DEPLOY_KEY_ID"
	EnvPRID          = "GITEA_PR_ID"
	EnvPRIndex       = "GITEA_PR_INDEX" // not used by Gitea at the moment, it is for custom git hooks
	EnvPushTrigger   = "GITEA_PUSH_TRIGGER"
	EnvIsInternal    = "GITEA_INTERNAL_PUSH"
	EnvAppURL        = "GITEA_ROOT_URL"
	EnvActionsTaskID = "GITEA_ACTIONS_TASK_ID"

	EnvRepoNameNew      = "FORGENTE_REPO_NAME"
	EnvRepoUsernameNew  = "FORGENTE_REPO_USER_NAME"
	EnvRepoIDNew        = "FORGENTE_REPO_ID"
	EnvRepoIsWikiNew    = "FORGENTE_REPO_IS_WIKI"
	EnvPusherNameNew    = "FORGENTE_PUSHER_NAME"
	EnvPusherEmailNew   = "FORGENTE_PUSHER_EMAIL"
	EnvPusherIDNew      = "FORGENTE_PUSHER_ID"
	EnvKeyIDNew         = "FORGENTE_KEY_ID"
	EnvDeployKeyIDNew   = "FORGENTE_DEPLOY_KEY_ID"
	EnvPRIDNew          = "FORGENTE_PR_ID"
	EnvPRIndexNew       = "FORGENTE_PR_INDEX"
	EnvPushTriggerNew   = "FORGENTE_PUSH_TRIGGER"
	EnvIsInternalNew    = "FORGENTE_INTERNAL_PUSH"
	EnvAppURLNew        = "FORGENTE_ROOT_URL"
	EnvActionsTaskIDNew = "FORGENTE_ACTIONS_TASK_ID"
)

// EnvGet reads a git-hook environment variable declared above: FORGENTE_* first, falling back
// to its legacy GITEA_* twin. Centralizes the read side so our own hook-handling code (cmd/hook.go)
// doesn't need to special-case each var.
func EnvGet(newName, oldName string) string {
	v, _ := setting.EnvWithFallback(newName, oldName)
	return v
}

// EnvPair renders a "NEW=value" and "OLD=value" pair for dual-emitting a hook env var under
// both its FORGENTE_* and legacy GITEA_* names with identical values.
func EnvPair(newName, oldName, value string) []string {
	return []string{newName + "=" + value, oldName + "=" + value}
}

type PushTrigger string

const (
	PushTriggerPRMergeToBase    PushTrigger = "pr-merge-to-base"
	PushTriggerPRUpdateWithBase PushTrigger = "pr-update-with-base"
)

// InternalPushingEnvironment returns an os environment to switch off hooks on push
// It is recommended to avoid using this unless you are pushing within a transaction
// or if you absolutely are sure that post-receive and pre-receive will do nothing
// We provide the full pushing-environment for other hook providers
func InternalPushingEnvironment(doer *user_model.User, repo *repo_model.Repository) []string {
	return append(PushingEnvironment(doer, repo),
		EnvPair(EnvIsInternalNew, EnvIsInternal, "true")...,
	)
}

// PushingEnvironment returns an os environment to allow hooks to work on push
func PushingEnvironment(doer *user_model.User, repo *repo_model.Repository) []string {
	return FullPushingEnvironment(doer, doer, repo, repo.Name, 0, 0)
}

func DoerPushingEnvironment(doer *user_model.User, repo *repo_model.Repository, isWiki bool) []string {
	env := []string{}
	env = append(env, EnvPair(EnvAppURLNew, EnvAppURL, setting.AppURL)...)
	env = append(env, EnvPair(EnvRepoNameNew, EnvRepoName, repo.Name+util.Iif(isWiki, ".wiki", ""))...)
	env = append(env, EnvPair(EnvRepoUsernameNew, EnvRepoUsername, repo.OwnerName)...)
	env = append(env, EnvPair(EnvRepoIDNew, EnvRepoID, strconv.FormatInt(repo.ID, 10))...)
	env = append(env, EnvPair(EnvRepoIsWikiNew, EnvRepoIsWiki, strconv.FormatBool(isWiki))...)
	env = append(env, EnvPair(EnvPusherNameNew, EnvPusherName, doer.Name)...)
	env = append(env, EnvPair(EnvPusherIDNew, EnvPusherID, strconv.FormatInt(doer.ID, 10))...)
	if !doer.KeepEmailPrivate {
		env = append(env, EnvPair(EnvPusherEmailNew, EnvPusherEmail, doer.Email)...)
	}
	if taskID, isActionsUser := user_model.GetActionsUserTaskID(doer); isActionsUser {
		env = append(env, EnvPair(EnvActionsTaskIDNew, EnvActionsTaskID, strconv.FormatInt(taskID, 10))...)
	}
	return env
}

// FullPushingEnvironment returns an os environment to allow hooks to work on push
func FullPushingEnvironment(author, committer *user_model.User, repo *repo_model.Repository, repoName string, prID, prIndex int64) []string {
	isWiki := strings.HasSuffix(repoName, ".wiki")
	authorSig := author.NewGitSig()
	committerSig := committer.NewGitSig()
	environ := append(os.Environ(),
		"GIT_AUTHOR_NAME="+authorSig.Name,
		"GIT_AUTHOR_EMAIL="+authorSig.Email,
		"GIT_COMMITTER_NAME="+committerSig.Name,
		"GIT_COMMITTER_EMAIL="+committerSig.Email,
		"SSH_ORIGINAL_COMMAND=gitea-internal",
	)
	environ = append(environ, EnvPair(EnvPRIDNew, EnvPRID, strconv.FormatInt(prID, 10))...)
	environ = append(environ, EnvPair(EnvPRIndexNew, EnvPRIndex, strconv.FormatInt(prIndex, 10))...)
	environ = append(environ, DoerPushingEnvironment(committer, repo, isWiki)...)
	return environ
}
