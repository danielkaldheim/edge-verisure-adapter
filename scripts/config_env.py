import sys

def set_deb_control(version , arch,file_name):
    template = "Package: verisure\n"
    template+= "Version: "+version+"\n"
    template+= "Replaces: verisure\n"
    template+= "Section: non-free/misc\n"
    template+= "Priority: optional\n"
    template+= "Architecture: "+arch+"\n"
    template+= "Maintainer: Daniel Rufus Kaldheim <daniel@kaldheim.org>\n"
    template+= "Description: . futurehome app  \n"

    f = open(file_name,"w")
    f.write(template)
    f.close()


def set_version_file(version):
    file_name = "./VERSION"
    f = open(file_name,"w")
    f.write(version)
    f.close()    


if __name__ == "__main__":
   environment = sys.argv[1] 
   version = sys.argv[2]
   arch = sys.argv[3]
   set_deb_control(version,arch,"./package/debian/DEBIAN/control")
   set_version_file(version)