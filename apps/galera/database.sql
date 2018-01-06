CREATE DATABASE menagerie;
USE menagerie;
CREATE TABLE pet (name VARCHAR(20), owner VARCHAR(20),species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);
INSERT INTO pet VALUES ('Puffball','Diane','hamster','f','1999-03-30',NULL);
CREATE USER 'galera'@'%' IDENTIFIED BY 'pass';
GRANT ALL PRIVILEGES ON *.* TO 'galera'@'%' WITH GRANT OPTION;

# SET PASSWORD FOR ''@'localhost' = PASSWORD('new_password');
# ALTER USER 'root'@'localhost' IDENTIFIED BY 'new_password';
# ALTER USER user IDENTIFIED BY 'new_password';
# CREATE USER 'galera'@'galera-q826r-svc' IDENTIFIED BY 'password';
# GRANT ALL PRIVILEGES ON *.* TO 'galera'@'galera-q826r-svc' WITH GRANT OPTION;
