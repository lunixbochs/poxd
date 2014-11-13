poxd
========

A personal TLS validation and interception proxy.

**Certificate validation is currently disabled. Do not trust outbound connections. You have been warned.**

Goals
--------

 - Proxy all outbound connections without adding any noticeable latency or throughput loss.
 - Enforce more strict CA policies than your system keychain.
    - This allows you to trust a corporate CA, but only for the corporate domains.
 - Bypass local application TLS bugs, and enforce certificate validation even if the app doesn't.
 - Intercept and modify your own traffic via an API.
