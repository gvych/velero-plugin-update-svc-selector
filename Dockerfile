# Copyright 2017, 2019, 2020 the Velero contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM --platform=linux/amd64 golang:1.22-bookworm AS build
ENV GOPROXY=https://proxy.golang.org
WORKDIR /go/src/github.com/eth-eks/velero-plugin-update-svc-selector
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/velero-plugin-update-svc-selector .

FROM --platform=linux/amd64 busybox:1.33.1 AS busybox

FROM --platform=linux/amd64 scratch
COPY --from=build /go/bin/velero-plugin-update-svc-selector /plugins/
COPY --from=busybox /bin/cp /bin/cp
USER 65532:65532
ENTRYPOINT ["/bin/cp", "/plugins/velero-plugin-update-svc-selector", "/target/"]
