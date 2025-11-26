<!--
SPDX-FileCopyrightText: 2025 Canonical Ltd
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
Copyright 2019 free5GC.org

SPDX-License-Identifier: Apache-2.0
-->
[![Go Report Card](https://goreportcard.com/badge/github.com/omec-project/nssf)](https://goreportcard.com/report/github.com/omec-project/nssf)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/omec-project/nssf/badge)](https://scorecard.dev/viewer/?uri=github.com/omec-project/nssf)

# NSSF

Compliance of the 5G Network functions can be found at [5G Compliance](https://docs.sd-core.opennetworking.org/main/overview/3gpp-compliance-5g.html)

## NSSF flow diagram
![NSSF Flow Diagram](/docs/images/README-NSSF.png)

## Dynamic Network configuration (via webconsole)

NSSF polls the webconsole every 5 seconds to fetch the latest PLMN configuration.

### Setting Up Polling

Include the `webuiUri` of the webconsole in the configuration file
```
configuration:
  ...
  webuiUri: https://webui:5001 # or http://webui:5001
  ...
```
The scheme (http:// or https://) must be explicitly specified. If no parameter is specified,
NSSF will use `http://webui:5001` by default.

### HTTPS Support

If the webconsole is served over HTTPS and uses a custom or self-signed certificate,
you must install the root CA certificate into the trust store of the NSSF environment.

Check the official guide for installing root CA certificates on Ubuntu:
[Install a Root CA Certificate in the Trust Store](https://documentation.ubuntu.com/server/how-to/security/install-a-root-ca-certificate-in-the-trust-store/index.html)

## Reach out to us through

1. #sdcore-dev channel in [Aether Project Slack](https://aether5g-project.slack.com)
2. Raise Github [issues](https://github.com/omec-project/nssf/issues/new)
