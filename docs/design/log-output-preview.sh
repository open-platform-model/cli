#!/usr/bin/env bash
# Preview script for the OPM log output style.
# Run: bash docs/design/log-output-preview.sh

# --- Colors ---
R="\033[0m"
DIM="\033[2m"
BOLD="\033[1m"
CYAN="\033[38;5;14m"
GREEN="\033[38;5;82m"
YELLOW="\033[38;5;220m"
RED="\033[38;5;196m"
BOLD_RED="\033[1;38;5;204m"
GREEN_CK="\033[38;5;10m"

# Level colors (charmbracelet/log defaults, MaxWidth 4)
LVL_INFO="\033[1;38;5;86m"
LVL_WARN="\033[1;38;5;192m"
LVL_ERR="\033[1;38;5;204m"
LVL_DEBUG="\033[1;38;5;63m"

# --- Helpers ---
ts()    { printf "${DIM}%s${R}" "$1"; }
info()  { printf "${LVL_INFO}INFO${R}"; }
erro()  { printf "${LVL_ERR}ERRO${R}"; }
warn()  { printf "${LVL_WARN}WARN${R}"; }
debu()  { printf "${LVL_DEBUG}DEBU${R}"; }
scope() { printf "${DIM}m:${R}${CYAN}%s${R} ${DIM}>${R}" "$1"; }

# Resource line: res <path> <status> <pad>
# All pads calculated so that len(path) + pad = 48
res() {
    local path="$1" status="$2" pad="$3"
    local sc
    case "$status" in
        created)    sc="${GREEN}created${R}" ;;
        configured) sc="${YELLOW}configured${R}" ;;
        unchanged)  sc="${DIM}unchanged${R}" ;;
        deleted)    sc="${RED}deleted${R}" ;;
        failed)     sc="${BOLD_RED}failed${R}" ;;
    esac
    printf "${DIM}r:${R}${CYAN}%s${R}%*s%b\n" "$path" "$pad" "" "$sc"
}

# Padding values (target: path + pad = 48)
# Namespace/production              20 → 28
# ServiceAccount/production/my-app  32 → 16
# ConfigMap/production/my-app-config 34 → 14
# Deployment/production/my-app      28 → 20
# Service/production/my-app         25 → 23

# --- Fresh Apply ---
echo ""
echo -e "${BOLD}━━━ Fresh Apply ━━━${R}"
echo ""

T="15:04:05"; M="my-app"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}applying${R} module ${CYAN}opm.dev/my-app${R} version 1.2.0"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}installing${R} ${CYAN}my-app${R} in namespace ${CYAN}production${R}"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Namespace/production"               created 28)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ServiceAccount/production/my-app"   created 16)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ConfigMap/production/my-app-config" created 14)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Deployment/production/my-app"       created 20)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Service/production/my-app"          created 23)"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}resources are ready${R}"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}applied successfully in 8s${R}"
echo -e "${GREEN_CK}✔${R} Module applied"

# --- Idempotent Re-run ---
echo ""
echo -e "${BOLD}━━━ Idempotent Re-run (partial change) ━━━${R}"
echo ""

T="15:05:12"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}applying${R} module ${CYAN}opm.dev/my-app${R} version 1.2.0"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}upgrading${R} ${CYAN}my-app${R} in namespace ${CYAN}production${R}"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Namespace/production"               unchanged 28)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ServiceAccount/production/my-app"   unchanged 16)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ConfigMap/production/my-app-config" configured 14)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Deployment/production/my-app"       unchanged 20)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Service/production/my-app"          unchanged 23)"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}resources are ready${R}"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}applied successfully in 3s${R}"
echo -e "${GREEN_CK}✔${R} Module applied"

# --- Delete ---
echo ""
echo -e "${BOLD}━━━ Delete ━━━${R}"
echo ""

T="15:06:30"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}deleting${R} ${CYAN}my-app${R} in namespace ${CYAN}production${R}"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Service/production/my-app"          deleted 23)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Deployment/production/my-app"       deleted 20)"
T="15:06:31"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ConfigMap/production/my-app-config" deleted 14)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "ServiceAccount/production/my-app"   deleted 16)"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Namespace/production"               deleted 28)"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}all resources have been deleted${R}"
echo -e "${GREEN_CK}✔${R} Module deleted"

# --- Error ---
echo ""
echo -e "${BOLD}━━━ Error ━━━${R}"
echo ""

T="15:07:00"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}applying${R} module ${CYAN}opm.dev/my-app${R} version 1.2.0"
echo -e "$(ts $T)  $(info) $(scope $M) ${BOLD}upgrading${R} ${CYAN}my-app${R} in namespace ${CYAN}production${R}"
echo -e "$(ts $T)  $(info) $(scope $M) $(res "Namespace/production"               unchanged 28)"
T="15:07:05"
echo -e "$(ts $T)  $(erro) $(scope $M) $(res "Deployment/production/my-app"       failed 20)"
echo -e "$(ts $T)  $(erro) $(scope $M) apply failed: context deadline exceeded"

# --- Debug ---
echo ""
echo -e "${BOLD}━━━ Debug (verbose mode) ━━━${R}"
echo ""

T="15:08:00"
echo -e "$(ts $T)  $(debu) $(scope $M) rendering module ${DIM}module${R}=${CYAN}opm.dev/my-app${R} ${DIM}namespace${R}=${CYAN}production${R}"
echo -e "$(ts $T)  $(debu) $(scope $M) loaded provider ${DIM}name${R}=${CYAN}kubernetes${R}"

# --- Warning ---
echo ""
echo -e "${BOLD}━━━ Warning ━━━${R}"
echo ""

T="15:08:00"
echo -e "$(ts $T)  $(warn) $(scope $M) resource ${CYAN}Deployment/production/my-app${R} has drift, will be reconciled"
echo ""
