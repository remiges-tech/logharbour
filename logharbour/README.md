### Installation of GeoLite2-City Database
To enable the functionality of the ListUnusualIPs client function, you'll need to install the GeoLite2-City Database from MaxMind. Follow the steps below to set it up:

**Step 1: Create an Account on MaxMind**
* Visit the MaxMind sign-up page:  [https://www.maxmind.com/en/geolite2/signup "MaxMind Account Signup"]
* Fill out the required information to create an account.

**Step 2: Download the GeoLite2-City Database**
* After creating an account, log in to the MaxMind website.
* Navigate to the Download Files section in the GeoIP2 / GeoLite2 category.
* Locate the GeoLite2 City database.
* Download the GeoLite2 City database in GZIP format.

**Step 3: Unzip and Place the Database File**
* Unzip the downloaded GZIP file. You will get a file named GeoLite2-City.mmdb.
* Copy the GeoLite2-City.mmdb file.
* Paste the GeoLite2-City.mmdb file into your project directory. For example, you might place it in a directory called data.

**Step 4: Configure config_dev.json**
* Open your project's config_dev.json file.
* Add or update the geolite_db_path parameter to point to the location of the GeoLite2-City.mmdb file.

 For example: config_dev.json
```
{
  "geolite_db_path": "./data/GeoLite2-City.mmdb",
  ...
}
```
* By following these steps, you will have installed the GeoLite2-City Database and configured your project to use it with the ListUnusualIPs function.
* This library is not specific to use only GeoLite2-City mmdb file ,it  supports all kinds of mmdb files.Ensure that the path in the configuration file points to the correct MMDB file you intend to use.



