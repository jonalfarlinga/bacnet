# Two subnets, so that the client can be tested as a true foreign device:
#
#   host (client)            foovm                         barvm
#   10.0.123.1  -- net A --  10.0.123.2 | 10.0.124.2  -- net B --  10.0.124.3
#                            IP router + bacserv (BBMD)             bacserv (device)
#                            BACnet bound to the net B side
#
# Net A is the private network the host is attached to. Net B is a VirtualBox
# internal network the host is not attached to, so its broadcasts never reach
# the client: the client sees barvm only through the BBMD on foovm. foovm
# forwards IP between the two, and barvm routes net A through it. The host
# needs one static route for net B via 10.0.123.2 (see the README).
Vagrant.configure("2") do |config|
    config.vm.box = "ubuntu/xenial64"
    config.ssh.forward_agent = true
    config.ssh.forward_x11 = true

    config.vm.synced_folder "../bacnet/", "/home/vagrant/bacnet"

    config.vm.provider "virtualbox" do |v|
        v.customize ["modifyvm", :id, "--memory", 1024]
    end

    config.vm.define "foovm" do |devvm|
        devvm.vm.hostname = 'foovm'
        devvm.vm.network :private_network, ip: "10.0.123.2"
        devvm.vm.network :private_network, ip: "10.0.124.2", virtualbox__intnet: "bacnet_b"
        devvm.vm.provision "shell", inline: <<-SHELL
            sysctl -w net.ipv4.ip_forward=1
            grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
        SHELL
    end

    config.vm.define "barvm" do |devvm|
        devvm.vm.hostname = 'barvm'
        devvm.vm.network :private_network, ip: "10.0.124.3", virtualbox__intnet: "bacnet_b"
        # Reach net A, and the host on it, through foovm.
        devvm.vm.provision "shell", run: "always", inline: <<-SHELL
            ip route replace 10.0.123.0/24 via 10.0.124.2
        SHELL
    end
end
