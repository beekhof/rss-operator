#! /bin/bash

set -eo pipefail

CFG=/etc/galera.peers
CFG_BAK=${CFG}.bak
MY_ID_FILE=/etc/galera.myid
HOSTNAME=$(hostname)

while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done

i=0
echo "" > "${CFG_BAK}"
for peer in "${PEERS[@]}"; do
    let i=i+1
    echo "server.${i}=${peer}" >> "${CFG_BAK}"
    if [[ "${peer}" == *"${HOSTNAME}"* ]]; then
      MY_ID=$i
      MY_NAME=${peer}
      echo $i > "${MY_ID_FILE}"
    fi
done

# zookeeper won't start without myid anyway.
# This means our hostname wasn't in the peer list.
if [ ! -f "${MY_ID_FILE}" ]; then
  exit 1
fi

# Once the dynamic config file is written it shouldn't be modified, so the final
# reconfigure needs to happen through the "reconfig" command.
cp ${CFG_BAK} ${CFG}
