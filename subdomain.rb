require 'socket'
require 'colorize'
begin
   domain = ARGV[0]
rescue
   puts "Usage: ruby subdomain.rb domain"
   exit
end
puts "+--------------------------------Creating Output Directory-------------------------------------+"
system("mkdir output/sub-domains/#{domain} || echo Directory Created")
system("mkdir output/sub-domains/#{domain}/amass || echo Directory Created")
system("mkdir output/sub-domains/#{domain}/censys || echo Directory Created")
system("mkdir output/sub-domains/#{domain}/subfinder || echo Directory Created")
system("mkdir output/sub-domains/#{domain}/wayback_url || echo Directory Created")
system("cp ./script.sh ./output/sub-domains/#{domain}/ || echo Script Created")


puts "+--------------------------------Subdomains By Subfinder-------------------------------------+"

system("subfinder -d #{domain} -o output/sub-domains/#{domain}/subfinder/#{domain}_subfinder.txt")
puts "Subfinder Ended..."
puts
puts "+--------------------------------Subdomains By Amass-------------------------------------+"

system("amass enum -timeout 30 -d #{domain} -o output/sub-domains/#{domain}/amass/#{domain}_amass.txt")
puts "Amass Ended..."
puts

puts "+--------------------------------Subdomains BY Censys-------------------------------------+"
system("python censys-subdomain-finder/censys_subdomain_finder.py #{domain} -o output/sub-domains/#{domain}/censys/#{domain}_censys.txt")
puts "Censys Ended..."




puts ".....................................Combining All domains........................................"
puts "Combining....."
system("cat output/sub-domains/#{domain}/amass/*.txt output/sub-domains/#{domain}/censys/*.txt output/sub-domains/#{domain}/subfinder/*.txt | sort -u > output/sub-domains/#{domain}/Combined_Domains.txt")
puts "................................... Looking for Wayback url ............................................"
system("cat ./output/sub-domains/#{domain}/Combined_Domains.txt | waybackurls > ./output/sub-domains/#{domain}/wayback_url/wayback_url.txt")
puts "................................... Creating Final List's ............................................"



  puts ENV['path']
  ENV['path'] = "/monster/tools/recon-my-way/output/sub-domains/#{domain}"
  system("sh ./output/sub-domains/#{domain}/script.sh")



  puts "..................................Number of ALL Subdomains (Combined_Domains.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/Combined_Domains.txt")

  puts "..................................Number of Responsive Subdomains (final_domain_list.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/final_domain_list.txt")

  puts "..................................Way Back Urls(wayback_url.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/wayback_url/wayback_url.txt")

  puts "..................................Way Back PHP Urls(phpurls.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/wayback_url/phpurls.txt")

  puts "..................................Way Back Urls(jsurls.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/wayback_url/jsurls.txt")

  puts "..................................Way Back Urls(jspurls.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/wayback_url/jspurls.txtl")

  puts "..................................Way Back Urls(aspxurls.txt)................................................"
  system("wc -l output/sub-domains/#{domain}/wayback_url/aspxurls.txt")
