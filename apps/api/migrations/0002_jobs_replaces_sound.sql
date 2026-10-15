-- Un job puede reemplazar el audio de un sonido existente: al terminar, el sonido nuevo toma su
-- lugar en el catálogo y en las bibliotecas, y el viejo se borra. Sin llave foránea: el sonido
-- viejo deja de existir justo cuando el job termina.
ALTER TABLE jobs ADD COLUMN replaces_sound_id TEXT;
