# Task 1: Nginx Web Server Setup

## Maqsad
Nginx web serverini o'rnatish va oddiy web sahifa yaratish

## Vazifa
1. Nginx web serverini o'rnating
2. `/var/www/html/index.html` faylini yarating  
3. Faylga `ok` matnini yozing
4. Nginx servisini ishga tushiring
5. Web sahifani tekshiring

## Ballar: 25 ball

---

## 💡 Maslahatlar

### Hint 1: Paketlarni yangilash
```bash
apt-get update
```

### Hint 2: Nginx o'rnatish
```bash
apt-get install -y nginx
```

### Hint 3: File yaratish
```bash
echo "ok" > /var/www/html/index.html
```

### Hint 4: Nginx ishga tushirish
```bash
service nginx start
```

### Hint 5: Tekshirish
```bash
curl http://127.0.0.1/index.html
```

---

## ✅ Tekshirish Mezonlari

1. `/var/www/html/index.html` fayli mavjud bo'lishi kerak - **5 ball**
2. Faylda `ok` matni bo'lishi kerak - **5 ball**
3. Nginx service ishlab turishi kerak - **10 ball**
4. Web sahifa ochilishi kerak - **5 ball**

## 🎯 Foydali Komandalar

- `ls /var/www/html` - fayllarni ko'rish
- `cat /var/www/html/index.html` - fayl mazmunini ko'rish
- `service nginx status` - nginx holatini tekshirish
