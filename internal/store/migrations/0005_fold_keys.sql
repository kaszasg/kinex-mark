-- Search and name-order keys without accents (kxm_fold, internal/store/driver.go):
-- rows written before this migration hold only lower-cased text.
UPDATE links
   SET search_text = kxm_fold(search_text),
       sort_label  = kxm_fold(sort_label)
 WHERE search_text <> kxm_fold(search_text)
    OR sort_label  <> kxm_fold(sort_label);
