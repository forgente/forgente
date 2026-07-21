#!/bin/bash

# FORGENTE_* is the primary env var name; GITEA_* is honored as a deprecated fallback for
# deployments still setting the old names (see Dockerfile.rootless ENV block for the defaults).
FORGENTE_WORK_DIR="${FORGENTE_WORK_DIR:-${GITEA_WORK_DIR:-/var/lib/forgente}}"
FORGENTE_CUSTOM="${FORGENTE_CUSTOM:-${GITEA_CUSTOM:-"$FORGENTE_WORK_DIR/custom"}}"
FORGENTE_TEMP="${FORGENTE_TEMP:-${GITEA_TEMP:-/tmp/forgente}}"
FORGENTE_APP_INI="${FORGENTE_APP_INI:-${GITEA_APP_INI:-/etc/forgente/app.ini}}"
export FORGENTE_WORK_DIR FORGENTE_CUSTOM FORGENTE_TEMP FORGENTE_APP_INI

# Prepare git folder
mkdir -p "${HOME}" && chmod 0700 "${HOME}"
if [ ! -w "${HOME}" ]; then echo "${HOME} is not writable"; exit 1; fi

# Prepare custom folder
mkdir -p "${FORGENTE_CUSTOM}" && chmod 0700 "${FORGENTE_CUSTOM}"

# Prepare temp folder
mkdir -p "${FORGENTE_TEMP}" && chmod 0700 "${FORGENTE_TEMP}"
if [ ! -w "${FORGENTE_TEMP}" ]; then echo "${FORGENTE_TEMP} is not writable"; exit 1; fi

#Prepare config file
if [ ! -f "${FORGENTE_APP_INI}" ]; then

    #Prepare config file folder
    FORGENTE_APP_INI_DIR=$(dirname "${FORGENTE_APP_INI}")
    mkdir -p "${FORGENTE_APP_INI_DIR}" && chmod 0700 "${FORGENTE_APP_INI_DIR}"
    if [ ! -w "${FORGENTE_APP_INI_DIR}" ]; then echo "${FORGENTE_APP_INI_DIR} is not writable"; exit 1; fi

    # Set INSTALL_LOCK to true only if SECRET_KEY is not empty and
    # INSTALL_LOCK is empty
    if [ -n "$SECRET_KEY" ] && [ -z "$INSTALL_LOCK" ]; then
        INSTALL_LOCK=true
    fi

    # Substitute the environment variables in the template
    APP_NAME=${APP_NAME:-"Gitea: Git with a cup of tea"} \
    RUN_MODE=${RUN_MODE:-"prod"} \
    RUN_USER=${USER:-"git"} \
    SSH_DOMAIN=${SSH_DOMAIN:-"localhost"} \
    HTTP_PORT=${HTTP_PORT:-"3000"} \
    ROOT_URL=${ROOT_URL:-""} \
    DISABLE_SSH=${DISABLE_SSH:-"false"} \
    SSH_PORT=${SSH_PORT:-"2222"} \
    SSH_LISTEN_PORT=${SSH_LISTEN_PORT:-} \
    DB_TYPE=${DB_TYPE:-"sqlite3"} \
    DB_HOST=${DB_HOST:-"localhost:3306"} \
    DB_NAME=${DB_NAME:-"gitea"} \
    DB_USER=${DB_USER:-"root"} \
    DB_PASSWD=${DB_PASSWD:-""} \
    INSTALL_LOCK=${INSTALL_LOCK:-"false"} \
    DISABLE_REGISTRATION=${DISABLE_REGISTRATION:-"false"} \
    REQUIRE_SIGNIN_VIEW=${REQUIRE_SIGNIN_VIEW:-"false"} \
    SECRET_KEY=${SECRET_KEY:-""} \
    envsubst < /etc/templates/app.ini > "${FORGENTE_APP_INI}"
fi

# Replace app.ini settings with env variables in the form FORGENTE__SECTION_NAME__KEY_NAME (or the legacy GITEA__ form)
environment-to-ini --config "${FORGENTE_APP_INI}"
