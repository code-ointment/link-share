# Fake a link coming up.
sudo ifconfig gpd0 up 10.180.89.252
sudo ip route add 10.0.0.0/8 via 10.180.89.252
