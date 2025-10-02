# Task 2: Docker Container Management

## Maqsad  
Docker containerlar bilan ishlashni o'rganish

## Vazifa
1. Docker xizmati ishlab turganini tekshiring
2. `hello-world` image'ini pull qiling
3. `hello-world` containerini ishga tushiring
4. `nginx:alpine` image'ini pull qiling
5. Ishlab turgan nginx containerini yarating

## Ballar: 30 ball

---

## 💡 Maslahatlar

### Hint 1: Docker xizmatini tekshirish
```bash
systemctl is-active docker
```

### Hint 2: Image pull qilish
```bash
docker pull hello-world
docker pull nginx:alpine
```

### Hint 3: Container ishga tushirish
```bash
docker run hello-world
```

### Hint 4: Background'da container ishga tushirish
```bash
docker run -d --name my-nginx nginx:alpine
```

### Hint 5: Containerlarni ko'rish
```bash
docker ps
```

---

## ✅ Tekshirish Mezonlari

1. Docker service active - **5 ball**
2. hello-world image mavjud - **5 ball**
3. hello-world container ishga tushgan - **5 ball**
4. nginx:alpine image mavjud - **5 ball**
5. Nginx container ishlab turibdi - **10 ball**

## 🎯 Foydali Komandalar

- `docker images` - imagelarni ko'rish
- `docker ps` - ishlab turgan containerlar
- `docker ps -a` - barcha containerlar
