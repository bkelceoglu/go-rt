# yaml dosyasinin validation bolumu

## yapilacaklar.

### format ve structure control

1. yaml dosyasi format olarak kontrol edilecek. (web ui tarafindan hazirlanan ya da kullanicinin hazirladigi)
2. aksayan bir taraf varsa, rapor edilecek. hata kullaniciya sunulacak. 
3. duzeltmeden sonra, format validation dogru olana kadar kullanici yeniden kontrol yapabilecek. 


### logic control 

1. bazi ataklarin, "attacker llm" gerektirmesi sebebiyle, eger kullanici attacker llm belirlemediyse, kullaniciya attacker llm isteyip istemedigi sorulacak. 
1.1. istemesi durumunda, ilgili alana attacker llm girmesi istenecek. 
1.2. istememesi durumunda: 
1.2.1. ilgili attack'larin cikartilmasi istenecek. 
1.2.2. judge llm ayni zamanda "attacker llm" olarak kullanilabilecek (sirali olmasi gerekli, atak sirasinda, judge yapilamaycak, judge sirasinda, atak yapilmayacak gibi)
		farkli bir attacker llm secilirse, siralamaya gerek olmayabilir. 

2. otomatik duzeltme: (her iki yontemin de verimliligi ve kullanim kosullarini onceden kontrol et ona gore devam edilsin)
2.1. mini bir engine yapilabilir 
2.2. judge llm tarafindan duzeltme istenebilir. 

otomatik duzeltme sonrasi kullanicidan onay alinacak. 

# bu feature hem cli hem de web ui icin yapilacak. 


