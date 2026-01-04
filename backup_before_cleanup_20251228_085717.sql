-- MySQL dump 10.13  Distrib 8.0.44, for Linux (x86_64)
--
-- Host: localhost    Database: realestate_db
-- ------------------------------------------------------
-- Server version	8.0.44

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `delete_logs`
--

DROP TABLE IF EXISTS `delete_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `delete_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `property_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` text COLLATE utf8mb4_unicode_ci,
  `detail_url` text COLLATE utf8mb4_unicode_ci,
  `removed_at` datetime DEFAULT NULL,
  `deleted_at` datetime NOT NULL,
  `reason` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_delete_logs_property_id` (`property_id`),
  KEY `idx_delete_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `delete_logs`
--

LOCK TABLES `delete_logs` WRITE;
/*!40000 ALTER TABLE `delete_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `delete_logs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `detail_scrape_queue`
--

DROP TABLE IF EXISTS `detail_scrape_queue`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `detail_scrape_queue` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `source` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source_property_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `detail_url` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `priority` bigint DEFAULT '0',
  `attempts` bigint DEFAULT '0',
  `last_error` text COLLATE utf8mb4_unicode_ci,
  `next_retry_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_queue_lookup` (`source`,`source_property_id`),
  KEY `idx_status` (`status`),
  KEY `idx_priority` (`priority`),
  KEY `idx_retry` (`next_retry_at`)
) ENGINE=InnoDB AUTO_INCREMENT=24 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `detail_scrape_queue`
--

LOCK TABLES `detail_scrape_queue` WRITE;
/*!40000 ALTER TABLE `detail_scrape_queue` DISABLE KEYS */;
INSERT INTO `detail_scrape_queue` VALUES (1,'yahoo','078650910173c5e27b70cbf9d288d12483ed693c','https://realestate.yahoo.co.jp/rent/detail/078650910173c5e27b70cbf9d288d12483ed693c','permanent_fail',0,3,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-21 19:36:56.016','2025-12-22 14:13:43.364','2025-12-22 14:13:43.363'),(2,'yahoo','07865091b34cde5443cdd4c176b833e5e1aa1fed','https://realestate.yahoo.co.jp/rent/detail/07865091b34cde5443cdd4c176b833e5e1aa1fed','permanent_fail',0,3,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-21 19:36:56.020','2025-12-22 14:18:49.749','2025-12-22 14:18:49.748'),(3,'yahoo','07736897f9e4932dd84f84a9e0b667f136e02ddb','https://realestate.yahoo.co.jp/rent/detail/07736897f9e4932dd84f84a9e0b667f136e02ddb','permanent_fail',0,3,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-21 19:36:56.022','2025-12-22 14:20:24.911','2025-12-22 14:20:24.910'),(4,'yahoo','077368975aa6231c6ab1e43e1feda1566b57d8c6','https://realestate.yahoo.co.jp/rent/detail/077368975aa6231c6ab1e43e1feda1566b57d8c6','permanent_fail',0,2,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-21 19:36:56.026','2025-12-22 14:21:34.086','2025-12-22 14:21:34.086'),(5,'yahoo','0731521967998b683879b453cd4fe030cdc18a9b','https://realestate.yahoo.co.jp/rent/detail/0731521967998b683879b453cd4fe030cdc18a9b','permanent_fail',0,2,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-21 19:36:56.028','2025-12-22 14:22:30.396','2025-12-22 14:22:30.395'),(6,'yahoo','07315219c6964da7a452a6f0bfeb69c548e4cc33','https://realestate.yahoo.co.jp/rent/detail/07315219c6964da7a452a6f0bfeb69c548e4cc33','processing',0,0,'',NULL,'2025-12-21 19:36:56.030','2025-12-21 19:36:56.063',NULL),(7,'yahoo','0416528326ef7832ca130f045d97deb668b1cf96','https://realestate.yahoo.co.jp/rent/detail/0416528326ef7832ca130f045d97deb668b1cf96','processing',0,0,'',NULL,'2025-12-21 19:36:56.031','2025-12-21 19:36:56.064',NULL),(8,'yahoo','041652832c06fb650f45760b3d1e384e914a521e','https://realestate.yahoo.co.jp/rent/detail/041652832c06fb650f45760b3d1e384e914a521e','processing',0,0,'',NULL,'2025-12-21 19:36:56.033','2025-12-21 19:36:56.065',NULL),(9,'yahoo','030335998a0e6c6996988c569bb84205c29282d0','https://realestate.yahoo.co.jp/rent/detail/030335998a0e6c6996988c569bb84205c29282d0','processing',0,0,'',NULL,'2025-12-21 19:36:56.035','2025-12-21 19:36:56.066',NULL),(10,'yahoo','011915755196bbe13cab189b7992d899e8418529','https://realestate.yahoo.co.jp/rent/detail/011915755196bbe13cab189b7992d899e8418529','processing',0,0,'',NULL,'2025-12-21 19:36:56.037','2025-12-21 19:36:56.068',NULL),(11,'yahoo','085443707917c88dec817608ea8f67b76bd04fc3','https://realestate.yahoo.co.jp/rent/detail/085443707917c88dec817608ea8f67b76bd04fc3','processing',0,0,'',NULL,'2025-12-21 19:36:56.039','2025-12-21 19:36:56.069',NULL),(12,'yahoo','0854437036bc2df8bf79b2e060ab212a9d054881','https://realestate.yahoo.co.jp/rent/detail/0854437036bc2df8bf79b2e060ab212a9d054881','processing',0,0,'',NULL,'2025-12-21 19:36:56.041','2025-12-21 19:36:56.070',NULL),(13,'yahoo','04180639509c9b18562d13313ce7f0ecce5b36a6','https://realestate.yahoo.co.jp/rent/detail/04180639509c9b18562d13313ce7f0ecce5b36a6','processing',0,0,'',NULL,'2025-12-21 19:36:56.043','2025-12-21 19:36:56.071',NULL),(14,'yahoo','0418063915de751322d19bbd53f86abe91735106','https://realestate.yahoo.co.jp/rent/detail/0418063915de751322d19bbd53f86abe91735106','processing',0,0,'',NULL,'2025-12-21 19:36:56.045','2025-12-21 19:36:56.073',NULL),(15,'yahoo','037821298604cf24d7c2a6b5e9c78d1ed0f64300','https://realestate.yahoo.co.jp/rent/detail/037821298604cf24d7c2a6b5e9c78d1ed0f64300','processing',0,0,'',NULL,'2025-12-21 19:36:56.047','2025-12-21 19:36:56.074',NULL),(16,'yahoo','0371614963597e232448944eb5a3aab3fe5d9a0a','https://realestate.yahoo.co.jp/rent/detail/0371614963597e232448944eb5a3aab3fe5d9a0a','processing',0,0,'',NULL,'2025-12-21 19:36:56.048','2025-12-21 19:36:56.077',NULL),(17,'yahoo','037161496a228c45b611e2343978a31e91cd0865','https://realestate.yahoo.co.jp/rent/detail/037161496a228c45b611e2343978a31e91cd0865','processing',0,0,'',NULL,'2025-12-21 19:36:56.049','2025-12-21 19:36:56.078',NULL),(18,'yahoo','03719629131f5c77b8d6de40ba26b896e5f5b4c8','https://realestate.yahoo.co.jp/rent/detail/03719629131f5c77b8d6de40ba26b896e5f5b4c8','processing',0,0,'',NULL,'2025-12-21 19:36:56.050','2025-12-21 19:36:56.079',NULL),(19,'yahoo','03941961f8fae7f392c76ba09706d6ba7e5115a2','https://realestate.yahoo.co.jp/rent/detail/03941961f8fae7f392c76ba09706d6ba7e5115a2','processing',0,0,'',NULL,'2025-12-21 19:36:56.052','2025-12-21 19:36:56.080',NULL),(20,'yahoo','03941961a40e57af9351e829761fd5f799dd9bf7','https://realestate.yahoo.co.jp/rent/detail/03941961a40e57af9351e829761fd5f799dd9bf7','processing',0,1,'failed to fetch URL: request failed after 3 retries: status code 404','2025-12-21 19:58:26.221','2025-12-21 19:36:56.053','2025-12-21 20:21:39.500',NULL),(21,'yahoo','08344802855382bec48cfcd6d29c5d0d56045b56','https://realestate.yahoo.co.jp/rent/detail/08344802855382bec48cfcd6d29c5d0d56045b56','permanent_fail',0,1,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-22 14:16:19.378','2025-12-22 15:13:05.377','2025-12-22 15:13:05.375'),(22,'yahoo','08344802863b9c759500e4151f91e1ce42139dc3','https://realestate.yahoo.co.jp/rent/detail/08344802863b9c759500e4151f91e1ce42139dc3','permanent_fail',0,1,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-22 14:16:19.383','2025-12-22 15:19:59.889','2025-12-22 15:19:59.888'),(23,'yahoo','08352518fe3a09a140fc9880fb125ddc9c968747','https://realestate.yahoo.co.jp/rent/detail/08352518fe3a09a140fc9880fb125ddc9c968747','permanent_fail',0,1,'404 Not Found (permanent): failed to fetch URL: permanent_fail: status code 404 (property not found or delisted)',NULL,'2025-12-22 14:16:19.386','2025-12-22 15:21:43.091','2025-12-22 15:21:43.090');
/*!40000 ALTER TABLE `detail_scrape_queue` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `properties`
--

DROP TABLE IF EXISTS `properties`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `properties` (
  `id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'yahoo',
  `source_property_id` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `detail_url` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `image_url` text COLLATE utf8mb4_unicode_ci,
  `rent` bigint DEFAULT NULL,
  `floor_plan` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `area` decimal(10,2) DEFAULT NULL,
  `walk_time` bigint DEFAULT NULL,
  `station` text COLLATE utf8mb4_unicode_ci,
  `address` text COLLATE utf8mb4_unicode_ci,
  `building_age` bigint DEFAULT NULL,
  `floor` bigint DEFAULT NULL,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `removed_at` datetime DEFAULT NULL,
  `last_seen_at` datetime DEFAULT NULL,
  `fetched_at` datetime NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_source_property` (`source`,`source_property_id`),
  KEY `idx_properties_detail_url` (`detail_url`),
  KEY `idx_properties_rent` (`rent`),
  KEY `idx_properties_floor_plan` (`floor_plan`),
  KEY `idx_properties_walk_time` (`walk_time`),
  KEY `idx_properties_status` (`status`),
  KEY `idx_properties_last_seen_at` (`last_seen_at`),
  KEY `idx_created_at` (`created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `properties`
--

LOCK TABLES `properties` WRITE;
/*!40000 ALTER TABLE `properties` DISABLE KEYS */;
INSERT INTO `properties` VALUES ('0de5e1642796136e650e03759e8f2c67','yahoo','a574df621c7fcee05b0427f01e85e958','https://realestate.yahoo.co.jp/rent/detail/_000003781314bc85d5f1e83280d9d80e82457182bf60','No Title','',NULL,'K',9.00,NULL,'','',NULL,86,'active',NULL,NULL,'2025-12-21 20:01:30','2025-12-21 19:28:39','2025-12-21 20:01:30'),('test_bracket_1','yahoo','tb1','https://example.com/b1','ãƒ†ã‚¹ãƒˆãƒžãƒ³ã‚·ãƒ§ãƒ³ã€Yahoo!ä¸å‹•ç”£ã€‘',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'active',NULL,NULL,'2025-12-28 08:57:03','2025-12-28 08:57:03','2025-12-28 08:57:03'),('test_bracket_2','yahoo','tb2','https://example.com/b2','ã€Yahoo!ä¸å‹•ç”£ã€‘ã‚µãƒ³ãƒ—ãƒ«ç‰©ä»¶',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'active',NULL,NULL,'2025-12-28 08:57:03','2025-12-28 08:57:03','2025-12-28 08:57:03'),('test_bracket_3','yahoo','tb3','https://example.com/b3','ã‚¢ãƒ‘ãƒ¼ãƒˆå ã€Yahooä¸å‹•ç”£ã€‘',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'active',NULL,NULL,'2025-12-28 08:57:03','2025-12-28 08:57:03','2025-12-28 08:57:03'),('test_bracket_4','yahoo','tb4','https://example.com/b4','é«˜ç´šãƒžãƒ³ã‚·ãƒ§ãƒ³ã€Yahooä¸å‹•ç”£ã€‘ABC',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'active',NULL,NULL,'2025-12-28 08:57:03','2025-12-28 08:57:03','2025-12-28 08:57:03');
/*!40000 ALTER TABLE `properties` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `property_changes`
--

DROP TABLE IF EXISTS `property_changes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `property_changes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `property_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `snapshot_id` bigint NOT NULL,
  `change_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `old_value` text COLLATE utf8mb4_unicode_ci,
  `new_value` text COLLATE utf8mb4_unicode_ci,
  `change_magnitude` decimal(10,2) DEFAULT NULL,
  `detected_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_property_changes_property_id` (`property_id`),
  KEY `idx_property_changes_detected_at` (`detected_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `property_changes`
--

LOCK TABLES `property_changes` WRITE;
/*!40000 ALTER TABLE `property_changes` DISABLE KEYS */;
/*!40000 ALTER TABLE `property_changes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `property_snapshots`
--

DROP TABLE IF EXISTS `property_snapshots`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `property_snapshots` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `property_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `snapshot_at` date NOT NULL,
  `rent` bigint DEFAULT NULL,
  `floor_plan` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `area` decimal(10,2) DEFAULT NULL,
  `walk_time` bigint DEFAULT NULL,
  `station` text COLLATE utf8mb4_unicode_ci,
  `address` text COLLATE utf8mb4_unicode_ci,
  `building_age` bigint DEFAULT NULL,
  `floor` bigint DEFAULT NULL,
  `image_url` text COLLATE utf8mb4_unicode_ci,
  `status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `has_changed` tinyint(1) DEFAULT '0',
  `change_note` text COLLATE utf8mb4_unicode_ci,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_property_date` (`snapshot_at`,`property_id`),
  KEY `idx_snapshot_date` (`snapshot_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `property_snapshots`
--

LOCK TABLES `property_snapshots` WRITE;
/*!40000 ALTER TABLE `property_snapshots` DISABLE KEYS */;
/*!40000 ALTER TABLE `property_snapshots` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-12-28  8:57:17
