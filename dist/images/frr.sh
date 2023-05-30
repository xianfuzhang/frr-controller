#!/bin/bash
#set -euo pipefail

# The argument to the command is the operation to be performed
# a cmd must be provided, there is no default
cmd=${1:-""}

frr_pod_name=${FRR_POD_NAME:-$(hostname)}
frrlogdir=/var/log/frr-controller

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
    -log_dir=${frrlogdir} \
    -log_file=${frrlogdir}/frr-controller.log

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
