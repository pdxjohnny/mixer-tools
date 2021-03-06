=====
mixer
=====

------------------------
OS software update mixer
------------------------

:Copyright: \(C) 2018 Intel Corporation, CC-BY-SA-3.0
:Manual section: 1


SYNOPSIS
========

``mixer [subcommand] <flags>``


DESCRIPTION
===========

``mixer``\(1) is a software update generator that uses input RPMs and bundle
definition files to create update metadata consumable by ``swupd``\(1).

``mixer`` operates in a mix workspace using configuration files existing in the
workspace. The tool allows users to use RPMs from an upstream URL (default
download.clearlinux.org), provide their own RPMs, and use RPMs from other remote
repositories to build updates. Users can also define their own local bundle
definitions as well as use the default bundle definitions from the upstream URL.

``mixer`` provides commands to manage local RPMs and bundle definitions and
configure remote RPM repositories.

The output of ``mixer`` is a set of manifests readable by ``swupd`` as well as
all the OS content ``swupd`` needs to perform its update operations. The OS
content includes all the files in an update as well as zero- and delta-packs for
improved update performance. The content that ``mixer`` produces is tied to a
specific format so that ``swupd`` is guaranteed to understand it if the client
is using the right version of ``swupd``. See ``swupd``\(1) and ``os-format``\(7)
for more details.


OPTIONS
=======

The following options are applicable to most subcommands, and can be
used to modify the core behavior and resources that ``mixer`` uses.

-  ``-h, --help``

   Display general help information. If put after a subcommand, it will
   display help specific to that subcommand.

-  ``--check``

   Check all external dependencies needed by mixer.

-  ``-v, --version``

   Displays the version information of the ``mixer`` program and exit.

-  ``--offline``

   Skip caching upstream bundles and work entirely with local bundles.
   Do not reach out over network to perform operations.

-  ``--native``

   Run mixer command on native host instead of in a container.


SUBCOMMANDS
===========

``add-rpms``

    Add RPMs from the configured LOCAL_RPM_DIR to the local dnf repository.
    See ``mixer.add-rpms``\(1) for more details.

``build``

    Build various pieces of OS content, including all metadata needed by
    ``swupd`` to perform updates. ``mixer build`` is a complicated tool in
    itself. See ``mixer.build``\(1) for more details.

``bundle``

    Perform various configuration actions on local and upstream bundles. The
    user can add or remove bundles from their mix, edit local and upstream
    bundles, create new bundle definitions, or validate local bundle definition
    files. See ``mixer.bundle``\(1) for more details.

``config``

    Perform configuration related actions, including configuration file
    validation and conversion from deprecated formats. See ``mixer.config``\(1)
    for more details.

``help``

    Print help text for any ``mixer`` subcommand.

``init``

    Initialize ``mixer`` configuration and workspace. See ``mixer.init``\(1) for
    more details.

``repo``

    Add, list, remove, or edit RPM repositories to be used by mixer. This
    subcommand allows users to configure which remote or local repositories
    mixer should use to look for RPMs. See ``mixer.repo``\(1) for more
    information.

``versions``

    Manage mix and upstream versions. By itself the command will print the
    current version of mix and upstream, and also report the latest version of
    upstream available. Also allows the user to update mix and upstream
    versions. See ``mixer.versions``\(1) for more details.


FILES
=====

`<mixer/workspace>/builder.conf`

    The mixer configuration file.

`<mixer/workspace>/.yum-mix.conf`

    The default location for the DNF configuration file.


EXIT STATUS
===========

On success, 0 is returned. A non-zero return code indicates a failure.

SEE ALSO
--------

* ``mixer.add-rpms``\(1)
* ``mixer.build``\(1)
* ``mixer.bundle``\(1)
* ``mixer.config``\(1)
* ``mixer.init``\(1)
* ``mixer.repo``\(1)
* ``mixer.versions``\(1)
* ``swupd``\(1)
* ``os-format``\(7)
* https://github.com/clearlinux/mixer-tools
* https://github.com/clearlinux/swupd-client
* https://clearlinux.org/documentation/
