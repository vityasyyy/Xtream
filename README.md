# xRabbit
Scalable software development KOM I Gede Mujiyatna. Scalable video streaming platform.

## Anggota Kelompok
1. Andrian Danar Perdana (23/513040/PA/21917)
2. Andreandhiki Riyanta Putra (23/517511/PA/22191)
3. Muhammad Argya Vityasy (23/522547/PA/22475) Kubernetes 
4. Nasya Putri Raudhah Dahlan (23/513931/PA/21967)

## Stress Testing

This project uses [k6](https://k6.io/) for stress testing.

### Prerequisites

1.  **Install k6**: Follow the instructions on the [k6 website](https://k6.io/docs/getting-started/installation/).
2.  **Ensure the application is running in Minikube**: Use the `./dev.sh` script.
3.  **Ensure `server.video.localhost` resolves to Minikube IP**:
    *   You might need to run `minikube tunnel` in a separate terminal.
    *   Alternatively, add an entry to your `/etc/hosts` file:
        ```
        <minikube_ip> server.video.localhost
        ```
        You can get `<minikube_ip>` by running `minikube ip`.

### Running Stress Tests

To run the stress tests, execute the following command from the `upload-service` directory:

```bash
./dev.sh k6
```

This will:
1. Execute the k6 script located at `scripts/stress_test.js`.

The k6 script will test the `/health` and `/videos` endpoints. You can modify the script in `scripts/stress_test.js` to change test parameters like duration, virtual users (VUs), and request rates.
