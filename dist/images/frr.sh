#!/bin/bash
#set -euo pipefail

# The argument to the command is the operation to be performed
# a cmd must be provided, there is no default
cmd=${1:-""}

frr_pod_name=${FRR_POD_NAME:-$(hostname)}
frrlogdir=/var/log/frr-controller

# Environment variables are used to customize operation
# VNI_RANGE - the vni allocation range
# ASN_RANGE - the asn allocation range for l2vpn
# LOGFILE_MAXSIZE - log file max size in MB(default 100 MB)
# LOGFILE_MAXBACKUPS - log file max backups (default 5)
# LOGFILE_MAXAGE - log file max age in days (default 5 days)

vni_range=${VNI_RANGE:-"1000-2000"}
asn_range=${ASN_RANGE:-"65001-65534"}

display_version() {
  echo " =================== Frr pod name: ${frr_pod_name}"
  if [[ -f /root/git_info ]]; then
    disp_ver=$(cat /root/git_info)
    echo " =================== Image built from frr-controller ${disp_ver}"
    return
  fi
}

# run frr controller
frr-controller() {
  echo "=============== frr-controller =============== "
  /usr/bin/frr-controller \
    --asn_range=${asn_range} \
    --vni_range=${vni_range} \
    --log_dir=${frrlogdir} \
    --log_file=${frrlogdir}/frr-controller.log

  echo "=============== frr-controller ========== running"
}


echo " ==================== command: ${cmd}"
display_version

case ${cmd} in
"frr-controller") # pod frr-controller container frr-controllers
  frr-controller
  ;;
*)
  echo "invalid command ${cmd}"
  echo "valid v3 commands: frr-controller"
  exit 0
  ;;
esac
