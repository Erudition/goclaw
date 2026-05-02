# Project Requirements & Constraints

## Avoid Redundant and Overlapping Volume Mounts
Ensure that overlay compose files (like `docker-compose.postgres.yml`) do not introduce legacy volume mounts or duplicate mounts to the same underlying volume that would break file isolation or overwrite specific image-defined directory ownership and permission layouts.
