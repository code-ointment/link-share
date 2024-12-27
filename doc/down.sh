# Fake a link going down.
sudo ip route del 10.0.0.0/8 via 10.180.89.252
sudo ifconfig gpd0 down 
