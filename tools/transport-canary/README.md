# Maintainer-only live transport canary

Explicit workflow_dispatch only. Creates one disposable free-trial account and
peer on the owned production VPN; sends no payments and creates no invoices.
The private configuration is kept in process memory and the probe stdin, never
logs or artifacts. A finally cleanup revokes the peer and session; a missing
cleanup acknowledgement makes the run fail. No global routes, host tunnels or
other VPN services are changed. The isolated userspace network uses the pinned
official AmneziaWG Go implementation plus instrumented DNS/TLS/HTTPS checks.

The six-second per-request deadline and sustained rekey verdict are fixed. The
report contains only safe counts, phase labels, public server IP and disposable
test IDs. This tests the Ubuntu runner network; it proves neither physical phone
import nor reachability from a particular Russia/Italy ISP. Qt key decoding is a
codec check, not execution of the native app. Workflow timeout/cancellation must
not be used to hide a failed check. Review cleanupConfirmed before rerunning.
