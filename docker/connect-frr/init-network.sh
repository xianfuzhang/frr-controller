#!/bin/bash
cmd=${1:-""}
vxlan_interface="vx"${VNI}
vxlan_vtep_local=${VTEP_LOCAL}
vni=${VNI}
bridge_name=br-vx${VNI}
internal_port_name=intp${VNI}
internal_iface_id=${SUBNET}-bm-l2gw

# create a linux bridge named $bridge_name, then add vxlan interface to it
if [ ! -d /sys/class/net/${bridge_name} ]; then
    ip link add ${bridge_name} type bridge
    ip link set ${bridge_name} up
fi

# create vxlan interface if not exist
if [ ! -d /sys/class/net/${vxlan_interface} ]; then
    ip link add ${vxlan_interface} type vxlan id ${vni} local ${vxlan_vtep_local} dstport 4789 nolearning
    ip link set ${vxlan_interface} up
fi


# add an internal port named ${internal_port_name} to ovs bridge br-int, and set the external_ids:iface-id to ${internal_iface_id}
ovs-vsctl --may-exist add-port br-int ${internal_port_name} -- set interface ${internal_port_name} type=internal external_ids:iface-id=${internal_iface_id}
ip link set ${internal_port_name} up

# add the internal port to the linux bridge
ip link set ${internal_port_name} master ${bridge_name}
# add the vxlan interface to the linux bridge
ip link set ${vxlan_interface} master ${bridge_name}



