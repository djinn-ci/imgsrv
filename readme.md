# imgsrv

imgsrv is the source code for the Djinn CI image server, that is hosted at
https://images.djinn-ci.com. This serves the base images for an installation
of Djinn CI, and provides a simple UI for viewing them. This watches the
location where base images are stored, and scans them into an in-memory database
at a set interval, which are then served.

The image server is configured using a simple configuration file, an example is
below,

    pidfile "/run/djinn/imgsrv.pid"
    
    log info "/var/log/djinn/imgsrv.log"
    
    net {
    	listen "localhost:8083"
    
    	write_timeout 10m
    	read_timeout  15s
    }
    
    store {
    	path "/var/lib/djinn/images/_base"
    
    	scan_interval 5m
    }
    
    driver qemu {
    	categories [
    		"x86_64",
    	]
    
    	groups [{
    		name    "Alpine"
    		pattern "alpine/*"
    	}, {
    		name    "Arch"
    		pattern "arch"
    	},{
    		name    "Debian"
    		pattern "debian/*"
    	}, {
    		name    "FreeBSD"
    		pattern "freebsd/*"
    	}, {
    		name    "Ubuntu"
    		pattern "ubuntu/*"
    	}]
    }

the `driver` block of the configuration is what handles the grouping and
categorization of images depending on the driver.

A driver category is any directory in the image location that is not considered
part of a driver's name. In the above example, there would be images organized
within the `x86_64` directory, which we would want categorized.

A driver group is a group of images whose names match a certain pattern.

The image server expects driver images to be organizes beneath the respective
directory for their driver, for example `qemu/` for the QEMU driver. Any
directory beneath that will either be treated as part of an image's name, or
a category for that image depending on how the image server is configured.

The above configuration is the exact configuration that is used to server
https://images.djinn-ci.com, if you want to see how it would render.
