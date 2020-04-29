
cat $path/wayback_url/wayback_url.txt | sort -u | unfurl --unique keys > $path/wayback_url/paramlist.txt
[ -s $path/wayback_url/paramlist.txt ] && echo "Wordlist saved to /wayback_url/paramlist.txt"

cat $path/wayback_url/wayback_url.txt  | sort -u | grep -P "\w+\.js(\?|$)" | sort -u > $path/wayback_url/jsurls.txt
[ -s $path/wayback_url/jsurls.txt ] && echo "JS Urls saved to $path/wayback_url/jsurls.txt"

cat $path/wayback_url/wayback_url.txt  | sort -u | grep -P "\w+\.php(\?|$)" | sort -u  > $path/wayback_url/phpurls.txt
[ -s $path/wayback_url/phpurls.txt ] && echo "PHP Urls saved to $path/wayback_url/phpurls.txt"

cat $path/wayback_url/wayback_url.txt  | sort -u | grep -P "\w+\.aspx(\?|$)" | sort -u  > $path/wayback_url/aspxurls.txt
[ -s $path/wayback_url/aspxurls.txt ] && echo "ASP Urls saved to $path/wayback_url/aspxurls.txt"

cat $path/wayback_url/wayback_url.txt  | sort -u | grep -P "\w+\.jsp(\?|$)" | sort -u  > $path/wayback_url/jspurls.txt
[ -s $path/wayback_url/jspurls.txt ] && echo "JSP Urls saved to $path/wayback_url/jspurls.txt"



echo "Probing for live hosts..."
cat $path/Combined_Domains.txt | sort -u | httprobe -c 50 -t 3000 >> $path/responsive.txt
cat $path/responsive.txt | sed 's/\http\:\/\///g' |  sed 's/\https\:\/\///g' | sort -u > $path/final_domain_list.txt
