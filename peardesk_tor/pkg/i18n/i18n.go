// Package i18n provides minimal UI string translations for 8 languages.
package i18n

import "sync"

// Supported language codes
const (
	IT = "it"
	EN = "en"
	FR = "fr"
	DE = "de"
	ES = "es"
	PT = "pt"
	RU = "ru"
	ZH = "zh"
)

var Languages = []string{IT, EN, FR, DE, ES, PT, RU, ZH}

var LangNames = map[string]string{
	IT: "Italiano",
	EN: "English",
	FR: "Français",
	DE: "Deutsch",
	ES: "Español",
	PT: "Português",
	RU: "Русский",
	ZH: "中文",
}

// strings holds all UI text keyed by [key][lang].
var strings = map[string]map[string]string{
	"app_title":         {IT: "PearDesk", EN: "PearDesk", FR: "PearDesk", DE: "PearDesk", ES: "PearDesk", PT: "PearDesk", RU: "PearDesk", ZH: "PearDesk"},
	"tab_connect":       {IT: "Connetti", EN: "Connect", FR: "Connecter", DE: "Verbinden", ES: "Conectar", PT: "Conectar", RU: "Подключить", ZH: "连接"},
	"tab_history":       {IT: "Cronologia", EN: "History", FR: "Historique", DE: "Verlauf", ES: "Historial", PT: "Histórico", RU: "История", ZH: "历史"},
	"tab_settings":      {IT: "Impostazioni", EN: "Settings", FR: "Paramètres", DE: "Einstellungen", ES: "Ajustes", PT: "Configurações", RU: "Настройки", ZH: "设置"},
	"your_pc":           {IT: "Il tuo PearDesk", EN: "Your PearDesk", FR: "Votre PearDesk", DE: "Ihr PearDesk", ES: "Su PearDesk", PT: "Seu PearDesk", RU: "Ваш PearDesk", ZH: "您的 PearDesk"},
	"your_id":           {IT: "Il tuo ID:", EN: "Your ID:", FR: "Votre ID :", DE: "Ihre ID:", ES: "Tu ID:", PT: "Seu ID:", RU: "Ваш ID:", ZH: "您的 ID："},
	"copy_id":           {IT: "Copia", EN: "Copy", FR: "Copier", DE: "Kopieren", ES: "Copiar", PT: "Copiar", RU: "Копировать", ZH: "复制"},
	"copied":            {IT: "Copiato!", EN: "Copied!", FR: "Copié !", DE: "Kopiert!", ES: "¡Copiado!", PT: "Copiado!", RU: "Скопировано!", ZH: "已复制！"},
	"regen_id":          {IT: "↺  Rigenera ID", EN: "↺  Regenerate ID", FR: "↺  Régénérer l'ID", DE: "↺  ID neu generieren", ES: "↺  Regenerar ID", PT: "↺  Regenerar ID", RU: "↺  Создать новый ID", ZH: "↺  重新生成 ID"},
	"regen_confirm":     {IT: "Generare un nuovo ID? Il vecchio non funzionerà più.", EN: "Generate a new ID? The old one will stop working.", FR: "Générer un nouvel ID ? L'ancien ne fonctionnera plus.", DE: "Neue ID generieren? Die alte hört auf zu funktionieren.", ES: "¿Generar un nuevo ID? El antiguo dejará de funcionar.", PT: "Gerar novo ID? O antigo deixará de funcionar.", RU: "Создать новый ID? Старый перестанет работать.", ZH: "生成新 ID？旧 ID 将失效。"},
	"access_password":   {IT: "Password accesso:", EN: "Access password:", FR: "Mot de passe :", DE: "Zugangskennwort:", ES: "Contraseña:", PT: "Senha de acesso:", RU: "Пароль доступа:", ZH: "访问密码："},
	"no_password":       {IT: "(nessuna password)", EN: "(no password)", FR: "(aucun mot de passe)", DE: "(kein Kennwort)", ES: "(sin contraseña)", PT: "(sem senha)", RU: "(без пароля)", ZH: "（无密码）"},
	"status_starting":   {IT: "Avvio in corso...", EN: "Starting...", FR: "Démarrage...", DE: "Wird gestartet...", ES: "Iniciando...", PT: "Iniciando...", RU: "Запуск...", ZH: "正在启动…"},
	"status_ready":      {IT: "Pronto! Il tuo ID:", EN: "Ready! Your ID:", FR: "Prêt ! Votre ID :", DE: "Bereit! Ihre ID:", ES: "¡Listo! Tu ID:", PT: "Pronto! Seu ID:", RU: "Готово! Ваш ID:", ZH: "就绪！您的 ID："},
	"status_stopped":    {IT: "Condivisione fermata", EN: "Sharing stopped", FR: "Partage arrêté", DE: "Freigabe gestoppt", ES: "Uso compartido detenido", PT: "Compartilhamento parado", RU: "Общий доступ остановлен", ZH: "共享已停止"},
	"connect_to_host":   {IT: "Connetti a un host", EN: "Connect to a host", FR: "Se connecter à un hôte", DE: "Mit einem Host verbinden", ES: "Conectar a un host", PT: "Conectar a um host", RU: "Подключиться к хосту", ZH: "连接到主机"},
	"id_host":           {IT: "ID Host:", EN: "Host ID:", FR: "ID de l'hôte :", DE: "Host-ID:", ES: "ID del host:", PT: "ID do host:", RU: "ID хоста:", ZH: "主机 ID："},
	"id_placeholder":    {IT: "ABC-123-XYZ", EN: "ABC-123-XYZ", FR: "ABC-123-XYZ", DE: "ABC-123-XYZ", ES: "ABC-123-XYZ", PT: "ABC-123-XYZ", RU: "ABC-123-XYZ", ZH: "ABC-123-XYZ"},
	"password":          {IT: "Password:", EN: "Password:", FR: "Mot de passe :", DE: "Kennwort:", ES: "Contraseña:", PT: "Senha:", RU: "Пароль:", ZH: "密码："},
	"if_required":       {IT: "(se richiesta)", EN: "(if required)", FR: "(si requis)", DE: "(falls erforderlich)", ES: "(si es necesario)", PT: "(se necessário)", RU: "(если требуется)", ZH: "（如需要）"},
	"remember_password": {IT: "Ricorda password per questo host", EN: "Remember password for this host", FR: "Mémoriser le mot de passe", DE: "Kennwort merken", ES: "Recordar contraseña", PT: "Lembrar senha", RU: "Запомнить пароль", ZH: "记住此主机的密码"},
	"connect_btn":       {IT: "  Connetti", EN: "  Connect", FR: "  Connecter", DE: "  Verbinden", ES: "  Conectar", PT: "  Conectar", RU: "  Подключить", ZH: "  连接"},
	"searching":         {IT: "Ricerca host", EN: "Searching for host", FR: "Recherche de l'hôte", DE: "Suche nach Host", ES: "Buscando host", PT: "Procurando host", RU: "Поиск хоста", ZH: "正在搜索主机"},
	"host_not_found":    {IT: "host non trovato", EN: "host not found", FR: "hôte introuvable", DE: "Host nicht gefunden", ES: "host no encontrado", PT: "host não encontrado", RU: "хост не найден", ZH: "未找到主机"},
	"enter_host_id":     {IT: "inserisci l'ID dell'host", EN: "please enter the host ID", FR: "veuillez entrer l'ID de l'hôte", DE: "Bitte die Host-ID eingeben", ES: "ingrese el ID del host", PT: "insira o ID do host", RU: "введите ID хоста", ZH: "请输入主机 ID"},
	"history_empty":     {IT: "Nessuna connessione nella cronologia.", EN: "No connections in history.", FR: "Aucune connexion dans l'historique.", DE: "Kein Verbindungsverlauf.", ES: "Sin historial de conexiones.", PT: "Sem histórico de conexões.", RU: "История подключений пуста.", ZH: "暂无连接历史。"},
	"connect":           {IT: "Connetti", EN: "Connect", FR: "Connecter", DE: "Verbinden", ES: "Conectar", PT: "Conectar", RU: "Подключить", ZH: "连接"},
	"remove":            {IT: "✕", EN: "✕", FR: "✕", DE: "✕", ES: "✕", PT: "✕", RU: "✕", ZH: "✕"},
	"startup_with_os":   {IT: "Avvia PearDesk all'avvio del PC", EN: "Start PearDesk with the OS", FR: "Démarrer avec le système", DE: "Mit dem System starten", ES: "Iniciar con el sistema", PT: "Iniciar com o sistema", RU: "Запускать при старте ОС", ZH: "随系统启动"},
	"language":          {IT: "Lingua / Language:", EN: "Language:", FR: "Langue :", DE: "Sprache:", ES: "Idioma:", PT: "Idioma:", RU: "Язык:", ZH: "语言："},
	"relay_url":         {IT: "URL Relay:", EN: "Relay URL:", FR: "URL Relay :", DE: "Relay-URL:", ES: "URL del relay:", PT: "URL do relay:", RU: "URL ретранслятора:", ZH: "中继 URL："},
	"save":              {IT: "Salva", EN: "Save", FR: "Enregistrer", DE: "Speichern", ES: "Guardar", PT: "Salvar", RU: "Сохранить", ZH: "保存"},
	"saved":             {IT: "Impostazioni salvate", EN: "Settings saved", FR: "Paramètres sauvegardés", DE: "Einstellungen gespeichert", ES: "Configuración guardada", PT: "Configurações salvas", RU: "Настройки сохранены", ZH: "设置已保存"},
	"connection":        {IT: "Connessione", EN: "Connection", FR: "Connexion", DE: "Verbindung", ES: "Conexión", PT: "Conexão", RU: "Подключение", ZH: "连接"},
	"files":             {IT: "File", EN: "Files", FR: "Fichiers", DE: "Dateien", ES: "Archivos", PT: "Arquivos", RU: "Файлы", ZH: "文件"},
	"connected_to":      {IT: "Connesso a", EN: "Connected to", FR: "Connecté à", DE: "Verbunden mit", ES: "Conectado a", PT: "Conectado a", RU: "Подключено к", ZH: "已连接到"},
	"connection_closed": {IT: "Connessione chiusa", EN: "Connection closed", FR: "Connexion fermée", DE: "Verbindung getrennt", ES: "Conexión cerrada", PT: "Conexão encerrada", RU: "Соединение закрыто", ZH: "连接已关闭"},
	"error":             {IT: "Errore", EN: "Error", FR: "Erreur", DE: "Fehler", ES: "Error", PT: "Erro", RU: "Ошибка", ZH: "错误"},
}

var (
	mu      sync.RWMutex
	current = IT
)

// SetLang sets the active language.
func SetLang(lang string) {
	mu.Lock()
	defer mu.Unlock()
	current = lang
}

// Lang returns the active language code.
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// T returns the translation of key in the current language,
// falling back to English, then the key itself.
func T(key string) string {
	mu.RLock()
	lang := current
	mu.RUnlock()
	if m, ok := strings[key]; ok {
		if s, ok := m[lang]; ok {
			return s
		}
		if s, ok := m[EN]; ok {
			return s
		}
	}
	return key
}
