## Cipher Worker Java

This is a Java implementation of the cipher worker, based on a Java Tasques client generated via the OpenApi generator.

### Running

Make sure the main server is running, and run

```shell
WORKER_ID=java-worker-1  ./gradlew run
```

The `WORKER_ID` env param is optional; it allows you to set the worker id used for making claims.