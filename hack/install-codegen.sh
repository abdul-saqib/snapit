#!/usr/bin/env bash
 
# A lightweight Kubernetes controller for automating PersistentVolumeClaim (PVC) snapshots.
# It allows users to create snapshots on-demand via custom resource policies or periodically
# according to a schedule.
#
# Copyright (C) 2025 Abdul Saqib
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

 
set -o errexit
set -o nounset
set -o pipefail

# ----------------------------
# Configuration
# ----------------------------
CODEGEN_REPO="git@github.com:kubernetes/code-generator.git"
CODEGEN_VERSION="v0.34.3"
CODEGEN_LOCAL_PATH="../code-generator"

# ----------------------------
# Install Kubernetes code-generator binaries
# ----------------------------
echo "Installing Kubernetes code-generator binaries..."
go install k8s.io/code-generator/cmd/client-gen@${CODEGEN_VERSION}
go install k8s.io/code-generator/cmd/lister-gen@${CODEGEN_VERSION}
go install k8s.io/code-generator/cmd/informer-gen@${CODEGEN_VERSION}
go install k8s.io/code-generator/cmd/deepcopy-gen@${CODEGEN_VERSION}
go install k8s.io/code-generator/cmd/defaulter-gen@${CODEGEN_VERSION}
go install k8s.io/code-generator/cmd/conversion-gen@${CODEGEN_VERSION}

# ----------------------------
# Clone code-generator repo if not present
# ----------------------------
if [[ ! -d "${CODEGEN_LOCAL_PATH}" ]]; then
    echo "Cloning Kubernetes code-generator repo into ${CODEGEN_LOCAL_PATH}..."
    git clone --branch ${CODEGEN_VERSION} "${CODEGEN_REPO}" "${CODEGEN_LOCAL_PATH}"
else
    echo "Kubernetes code-generator repo already exists at ${CODEGEN_LOCAL_PATH}, skipping clone."
fi

echo "Setup complete!"
