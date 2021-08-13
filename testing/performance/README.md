
# Load Test

Rate User

```bash
$wrk -t16 -c20 -d30s -s ./testing/rateUser.lua http://127.0.0.1/ratings
```

Leader Board User Rating

```bash
$wrk -t16 -c16 -d30s http://127.0.0.1/users/ratings
```
