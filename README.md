# Hyprspace
<img src="hyprspace.png" width="300" height="300">
Hyprspace is a Wireguard-like VPN built on top of Libp2p. 

## What is Libp2p and Why Use it?
[Libp2p](libp2p.io) is a networking library created by [Protocol Labs](https://protocol.ai) that allows nodes to discover eachother using a Distributed Hash Table. Paired with [NAT hole punching](https://en.wikipedia.org/wiki/Hole_punching_(networking)) this allows Hyprspace to create a direct encrypted tunnel between two nodes even if they're both behind firewalls.

**Moreover! Each node doesn't even need to know the other's ip address prior to starting up the connection.** This makes Hyprspace perfect for devices that frequently migrate between locations but still require a constant virtual ip address.

## Use Cases:
##### A Digital Nomad
I use this system when travelling, if I'm staying in a rental or hotel and want to try something out on a Raspberry Pi I can plug the Pi into the location's router or ethernet port and then just ssh into the system using the same-old internal Hyprspace ip address without having to worry about their NAT or local firewall. Furthermore, if I'm connected to the Virtual Hyprspace Network I can ssh into my machines at home without requiring me to set up any sort of port forwarding.

##### A Privacy Advocate
Honestly, I even use this system when I'm at home and could connect directly to my local infastructure. Using Hyprspace however, I don't have to trust the security of my local network and hyprspace will intelligently connect to my machines using their local ip addresses for maximum speed.

If anyone else has some use cases please add them! Pull requests welcome!

## Prerequisites
If you're running Hyprspace on a Mac you'll need to install `iproute2mac`. If you're using the `brew` package manager that's as simple as,
```bash
brew install iproute2mac
```

## Installation

1. Go to Hyprspace Releases (other there -->)
2. Copy the link for your corresponding OS and Architecture.
3. Run `sudo curl -L "PATH-TO-RELEASE" -o /usr/local/bin/hyprspace`
4. Run `sudo chmod a+x /usr/local/bin/hyprspace`
5. (Optional) Run `sudo ln -s /usr/local/bin/hyprspace /usr/bin/hyprspace`

## Usage

### Commands

| Command             |  Alias  | Description                                                                |
| ------------------- | ------- | -------------------------------------------------------------------------- |
| `help`              | `?`     | Get help with a specific subcommand.                                       |
| `init`              | `i`     | Initialize an interface's configuration.                                   |
| `up`                | `up`    | Create and Bring Up a Hyprspace Interface                                  |
| `down  `            | `d`     | Bring Down and Delete A Hyprspace Interface                                |
| `update`            | `upd`   | Have Hyprspace update its own binary.                                      |

## Tutorial

### Initializing an Interface

The first thing we'll want to do once we've got Hyprspace installed is
initialize the configuration for an interface. In this case we'll call the
interface on our local machine `hs0` (for hypr-space 0) and `hs1` on our remote server
but yours could be anything you'd like. 

(Note: if you're using a Mac you'll have to use the interface name `utun[0-9]`. Check which interfaces are already in use by running `ip a` once you've got `iproute2mac` installed.)

###### Local Machine
```bash
sudo hyprspace init hs0
```

###### Remote Machine
```bash
sudo hyprspace init hs1
```

### Add Each Machine As A Peer Of The Other

Now that we've got a set of configurations we'll want to
tell the machines about eachother. By default Hyprspace will
put the interface configurations in `/etc/hyprspace/interface-name.yaml`.
So for our example we'll run

###### Local Machine
```bash
sudo nano /etc/hyprspace/hs0.yaml
```

and

###### Remote Machine
```bash
sudo nano /etc/hyprspace/hs1.yaml
```

### Update Peer Configs

Now in each config we'll add the other machine's ID as a peer.
You can find each machine's ID at the top of their configuration file.
Update,

```yaml
peers: {}
```
to 
```yaml
peers:
  10.1.1.2:
    id: YOUR-OTHER-PEER-ID
```

Notice here we'll have to pick one of our machines to be `10.1.1.1` 
and the other to be `10.1.1.2`. Make sure to update the interface's IP
address for the machine who needs to change to be `10.1.1.2`.

### Update our Discover Key

Looking in the interface's configuration you'll also notice a field called
`discover_key` (right above the interface's private key). It doesn't matter
which discovery key you pick but it much be the same for all of the nodes in your little cluster
so that they can find eachother.

(*Note you can use different `discover_key`s with different interfaces on the same 
host to create different isolated networks.)

```yaml
discover_key: fiftieth-dandelion-wronged-craftwork
```

### Starting Up the Interfaces!
Now that we've got our configs all sorted we can start up the two interfaces!

###### Local Machine
```bash
sudo hyprspace up hs0
```

and

###### Remote Machine
```bash
sudo hyprspace up hs1
```

After a few seconds you should see a the network finish setting up
and find your other machine. We can now test the connection by
pinging back and forth across the network.

###### Local Machine
```bash
ping 10.1.1.2
```

### Stoping the Interface and Cleaning Up
Now to stop the interface and clean up the system you can run,

###### Local Machine
```bash
sudo hyprspace down hs0
```

and,

###### Remote Machine
```bash
sudo hyprspace down hs1
```

## License

Copyright 2021 Alec Scott <hi@alecbcs.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
